package spotify

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/amonks/genres/data"
	"github.com/amonks/genres/request"
)

const nextReqFilename = "next-req"

// New creates a new Spotify client, with the given clientID and clientSecret.
func New(clientID, clientSecret string) *Client {
	var nextReqAt time.Time
	if _, err := os.Stat(nextReqFilename); !errors.Is(err, os.ErrNotExist) {
		bs, err := os.ReadFile(nextReqFilename)
		if err != nil {
			panic(err)
		}
		nextReqAt, err = time.Parse(time.UnixDate, string(bs))
		if err != nil {
			panic(err)
		}
	}
	return &Client{
		clientID:     clientID,
		clientSecret: clientSecret,
		nextReqAt:    nextReqAt,
		delay:        time.Second / 10,
	}
}

type Client struct {
	clientID     string
	clientSecret string

	nextReqAt time.Time
	delay     time.Duration

	accessToken string
	expiresAt   time.Time
}

// FetchGenre does a search for artists with the given genre, fetches the first
// 20 pages of search results, and returns up to 1000 artists.
//
// The request is basically,
//
//	https://api.spotify.com/v1/search?query=genre:GENRE&type=artist
//
// FetchGenre respects Spotify's documented semantics around its rate limiter:
// checking for a Retry-After header when it receives a 429 response. If
// FetchGenre is rate limited, it won't error, but it might take a long time.
func (spo *Client) FetchGenre(ctx context.Context, name string) ([]data.Artist, error) {
	var artists []data.Artist
	for offset := 0; offset < 1000; offset += 50 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		resp, err := spo.fetchGenrePage(ctx, name, offset)
		if err != nil {
			return nil, err
		}
		for _, item := range resp.Artists.Items {
			var imageURL string
			var maxSize int64
			for _, image := range item.Images {
				if image.Width > maxSize {
					imageURL = image.URL
				}
			}
			hasOriginalGenre := false
			for _, genre := range item.Genres {
				if genre == name {
					hasOriginalGenre = true
					break
				}
			}
			if !hasOriginalGenre {
				item.Genres = append(item.Genres, name)
			}
			artists = append(artists, data.Artist{
				SpotifyID:  item.ID,
				Name:       item.Name,
				ImageURL:   imageURL,
				Followers:  item.Followers.Total,
				Popularity: item.Popularity,
				Genres:     item.Genres,
			})
		}

		// intentionally not respecting the "next: null" pagination
		// thing here, because,
		// - everynoise.com doesn't respect it either, and,
		// - I often get a "next: null" even when there are many more
		//   result pages I can request successfully
		if len(resp.Artists.Items) < 50 {
			break
		}
	}
	return artists, nil
}

func (spo *Client) fetchGenrePage(ctx context.Context, name string, offset int) (*genreSearchResults, error) {
retry:
	if !spo.nextReqAt.IsZero() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		log.Printf("next request in %s", spo.nextReqAt.Sub(time.Now()).Truncate(time.Second))
	wait:
		for {
			select {
			case now := <-ticker.C:
				log.Printf("next request in %s", spo.nextReqAt.Sub(now).Truncate(time.Second))
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Until(spo.nextReqAt)):
				break wait
			}
		}
		if err := os.Remove(nextReqFilename); err != nil && !errors.Is(err, os.ErrNotExist) {
			panic(err)
		}
	}

	url, _ := url.Parse("https://api.spotify.com/v1/search")
	query := url.Query()
	query.Add("query", fmt.Sprintf(`genre:"%s"`, name))
	query.Add("type", "artist")
	query.Add("limit", "50")
	query.Add("offset", fmt.Sprintf("%d", offset))
	url.RawQuery = query.Encode()
	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("genre request error: %w", err)
	}

	token, err := spo.token()
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("genre request error: %w", err)
	}
	if resp.StatusCode == 429 {
		spo.delay = 2 * spo.delay
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter == "" {
			log.Printf("no retry-after header on 429; retrying in 1 minute")
			spo.nextReqAt = time.Now().Add(time.Minute)
		} else {
			seconds, err := strconv.ParseInt(retryAfter, 10, 64)
			if err != nil {
				return nil, err
			}
			waitTime := time.Duration(seconds)*time.Second + time.Second
			log.Printf("429; retrying in %s", waitTime)
			spo.nextReqAt = time.Now().Add(waitTime)
		}
		if err := os.WriteFile(nextReqFilename, []byte(spo.nextReqAt.Format(time.UnixDate)), 0666); err != nil {
			return nil, err
		}
		goto retry
	}
	defer resp.Body.Close()
	if err := request.Error(resp); err != nil {
		return nil, fmt.Errorf("genre fetch error: %w", err)
	}

	var results genreSearchResults
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&results); err != nil {
		return nil, fmt.Errorf("genre decode error: %w", err)
	}

	spo.nextReqAt = time.Now().Add(spo.delay)

	return &results, nil
}

type genreSearchResults struct {
	Artists struct {
		Limit  int
		Offset int
		Total  int

		Next     string
		Previous string

		Items []struct {
			Followers struct {
				Total int64
			}
			Genres []string
			ID     string
			Images []struct {
				Height int64
				Width  int64
				URL    string
			}
			Name       string
			Popularity int64
		}
	}
}
type tokenResult struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

func (spo *Client) token() (string, error) {
	if spo.accessToken == "" || spo.expiresAt.Before(time.Now().Add(time.Second)) {
		if err := spo.fetchToken(); err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("Bearer %s", spo.accessToken), nil
}

func (spo *Client) fetchToken() error {
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	url := "https://accounts.spotify.com/api/token"
	req, err := http.NewRequest("POST", url, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("token request error: %w", err)
	}
	up := fmt.Sprintf("%s:%s", spo.clientID, spo.clientSecret)
	credential := base64.StdEncoding.EncodeToString([]byte(up))
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", credential))
	req.Header.Set("Content-type", "application/x-www-form-urlencoded")

	requestAt := time.Now()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("token request error: %w", err)
	}
	defer resp.Body.Close()
	if err := request.Error(resp); err != nil {
		return fmt.Errorf("token fetch error: %w", err)
	}

	var result tokenResult
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&result); err != nil {
		return fmt.Errorf("token decode error: %w", err)
	}

	spo.accessToken = result.AccessToken
	spo.expiresAt = requestAt.Add(time.Duration(result.ExpiresIn) * time.Second)

	return nil
}
