package spotify

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
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

	client := &Client{
		clientID:     clientID,
		clientSecret: clientSecret,
		nextReqAtPtr: atomic.Pointer[time.Time]{},
		delay:        time.Second / 10,
	}
	client.setNextReqAt(nextReqAt)
	return client
}

type Client struct {
	mu sync.Mutex

	clientID     string
	clientSecret string

	nextReqAtPtr atomic.Pointer[time.Time]
	delay        time.Duration

	accessToken string
	expiresAt   time.Time
}

func (spo *Client) FetchAlbums(ctx context.Context, albumSpotifyIDs []string) ([]data.Album, error) {
	query := url.Values{}
	query.Add("ids", strings.Join(albumSpotifyIDs, ","))

	resp, err := spo.get(ctx, "https://api.spotify.com/v1/albums", query)
	if err != nil {
		return nil, err
	}

	defer resp.Close()
	var results albumsTracks
	dec := json.NewDecoder(resp)
	if err := dec.Decode(&results); err != nil {
		return nil, fmt.Errorf("albums tracks decode error: %w", err)
	}

	var albums []data.Album

	for _, fetched := range results.Albums {
		album := data.Album{
			SpotifyID:            fetched.ID,
			Name:                 fetched.Name,
			Type:                 fetched.AlbumType,
			ImageURL:             fetched.Images[0].URL,
			TotalTracks:          fetched.TotalTracks,
			HasFetchedTracks:     true,
			ReleaseDate:          fetched.ReleaseDate,
			ReleaseDatePrecision: fetched.ReleaseDatePrecision,
			Artists:              make([]data.Artist, len(fetched.Artists)),
			Tracks:               make([]data.Track, fetched.Tracks.Total),
		}
		for _, track := range fetched.Tracks.Items {
			artists := make([]data.Artist, len(track.Artists))
			for i, artist := range track.Artists {
				artists[i] = data.Artist{
					SpotifyID: artist.ID,
					Name:      artist.Name,
				}
			}
			album.Tracks = append(album.Tracks, data.Track{
				SpotifyID:  track.ID,
				Name:       track.Name,
				Popularity: track.Popularity,

				AlbumSpotifyID: fetched.ID,
				AlbumName:      fetched.Name,
				DiscNumber:     track.DiscNumber,
				TrackNumber:    track.TrackNumber,
				Artists:        artists,
			})
		}
		if len(fetched.Tracks.Items) <= fetched.Tracks.Total {
			continue
		}
		for offset := 50; offset < 1000; offset += 50 {
			query := url.Values{}
			query.Add("limit", "50")
			query.Add("offset", fmt.Sprintf("%d", offset))
			resp, err := spo.get(ctx, fmt.Sprintf("https://api.spotify.com/v1/albums/%s/tracks", fetched.ID), query)
			if err != nil {
				return nil, err
			}

			defer resp.Close()
			var results albumTracksPage
			dec := json.NewDecoder(resp)
			if err := dec.Decode(&results); err != nil {
				return nil, err
			}

			for _, track := range results.Items {
				artists := make([]data.Artist, len(track.Artists))
				for i, artist := range track.Artists {
					artists[i] = data.Artist{
						SpotifyID: artist.ID,
						Name:      artist.Name,
					}
				}
				album.Tracks = append(album.Tracks, data.Track{
					SpotifyID:  track.ID,
					Name:       track.Name,
					Popularity: track.Popularity,

					AlbumSpotifyID: fetched.ID,
					AlbumName:      fetched.Name,
					DiscNumber:     track.DiscNumber,
					TrackNumber:    track.TrackNumber,
					Artists:        artists,
				})
			}

			if results.Next == "" {
				break
			}
		}
	}
	return albums, nil
}

type albumsTracks struct {
	Albums []struct {
		AlbumType   string
		TotalTracks int64
		ID          string
		Images      []struct {
			URL string
		}
		Name                 string
		ReleaseDate          string
		ReleaseDatePrecision string
		Artists              []struct {
			Name string
			ID   string
		}
		Tracks struct {
			Limit  int
			Offset int
			Total  int

			Next     string
			Previous string

			Items []struct {
				ID         string
				Name       string
				Popularity int64

				DiscNumber  int64
				TrackNumber int64

				Artists []struct {
					ID   string
					Name string
				}
			}
		}
	}
}

func (spo *Client) fetchAlbumTracksPage(ctx context.Context, albumSpotifyID string, offset int) (*albumTracksPage, error) {
	query := url.Values{}
	query.Add("limit", "50")
	query.Add("offset", fmt.Sprintf("%d", offset))

	resp, err := spo.get(ctx, fmt.Sprintf("https://api.spotify.com/v1/albums/%s/tracks", albumSpotifyID), query)
	if err != nil {
		return nil, err
	}

	defer resp.Close()
	var results albumTracksPage
	dec := json.NewDecoder(resp)
	if err := dec.Decode(&results); err != nil {
		return nil, fmt.Errorf("album tracks decode error: %w", err)
	}

	return &results, nil
}

type albumTracksPage struct {
	Limit  int
	Offset int
	Total  int

	Next     string
	Previous string

	Items []struct {
		ID         string
		Name       string
		Popularity int64

		DiscNumber  int64
		TrackNumber int64

		Artists []struct {
			ID   string
			Name string
		}
	}
}

func (spo *Client) FetchArtistAlbums(ctx context.Context, artistSpotifyID string) ([]data.Album, error) {
	var albums []data.Album
	for offset := 0; offset < 1000; offset += 50 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		resp, err := spo.fetchArtistAlbumsPage(ctx, artistSpotifyID, offset)
		if err != nil {
			return nil, err
		}
		for _, album := range resp.Items {
			var imageURL string
			if len(album.Images) > 0 {
				imageURL = album.Images[0].URL
			}
			artists := make([]data.Artist, len(album.Artists))
			for i, artist := range album.Artists {
				artists[i] = data.Artist{
					SpotifyID: artist.ID,
					Name:      artist.Name,
				}
			}
			albums = append(albums, data.Album{
				SpotifyID:            album.ID,
				Name:                 album.Name,
				Type:                 album.AlbumType,
				ImageURL:             imageURL,
				TotalTracks:          album.TotalTracks,
				HasFetchedTracks:     false,
				ReleaseDate:          album.ReleaseDate,
				ReleaseDatePrecision: album.ReleaseDatePrecision,
				Artists:              artists,
			})
		}
		if len(resp.Items) < 50 {
			break
		}
	}
	return albums, nil
}

func (spo *Client) fetchArtistAlbumsPage(ctx context.Context, artistSpotifyID string, offset int) (*artistAlbumsPage, error) {
	query := url.Values{}
	query.Add("limit", "50")
	query.Add("offset", fmt.Sprintf("%d", offset))
	query.Add("include_groups", "album,single")

	resp, err := spo.get(ctx, fmt.Sprintf("https://api.spotify.com/v1/artists/%s/albums", artistSpotifyID), query)
	if err != nil {
		return nil, err
	}

	defer resp.Close()
	var results artistAlbumsPage
	dec := json.NewDecoder(resp)
	if err := dec.Decode(&results); err != nil {
		return nil, fmt.Errorf("artist albums decode error: %w", err)
	}

	return &results, nil
}

type artistAlbumsPage struct {
	Limit  int
	Offset int
	Total  int

	Next     string
	Previous string

	Items []struct {
		AlbumType   string
		TotalTracks int64
		ID          string
		Images      []struct {
			URL string
		}
		Name                 string
		ReleaseDate          string
		ReleaseDatePrecision string
		Artists              []struct {
			Name string
			ID   string
		}
	}
}

func (spo *Client) FetchTrackAnalyses(ctx context.Context, ids []string) ([]data.Track, error) {
	query := url.Values{}
	query.Set("ids", strings.Join(ids, ","))
	resp, err := spo.get(ctx, "https://api.spotify.com/v1/audio-features", query)
	if err != nil {
		return nil, err
	}

	defer resp.Close()

	var results trackAnalysesResults
	dec := json.NewDecoder(resp)
	if err := dec.Decode(&results); err != nil {
		return nil, fmt.Errorf("track analysis decode error: %w", err)
	}

	if len(results.AudioFeatures) != len(ids) {
		return nil, fmt.Errorf("expected %d analyses but got %d", len(ids), len(results.AudioFeatures))
	}

	var tracks []data.Track
	emptyTracks := 0
	for _, track := range results.AudioFeatures {
		if track.ID == "" {
			emptyTracks++
			continue
		}
		tracks = append(tracks, data.Track{
			SpotifyID: track.ID,

			Key:           track.Key,
			Mode:          track.Mode,
			Tempo:         track.Tempo,
			TimeSignature: track.TimeSignature,

			Acousticness:     track.Acousticness,
			Danceability:     track.Danceability,
			Energy:           track.Energy,
			Instrumentalness: track.Instrumentalness,
			Liveness:         track.Liveness,
			Loudness:         track.Loudness,
			Speechiness:      track.Speechiness,
			Valence:          track.Valence,
		})
	}

	return tracks, nil
}

type trackAnalysesResults struct {
	AudioFeatures []struct {
		ID string

		Key           int64
		Mode          int64
		Tempo         float64
		TimeSignature int64

		Acousticness     float64
		Danceability     float64
		Energy           float64
		Instrumentalness float64
		Liveness         float64
		Loudness         float64
		Speechiness      float64
		Valence          float64
	} `json:"audio_features"`
}

func (spo *Client) FetchArtistTracks(ctx context.Context, artistID string) ([]data.Track, error) {
	resp, err := spo.get(ctx, fmt.Sprintf("https://api.spotify.com/v1/artists/%s/top-tracks", artistID), nil)
	if err != nil {
		return nil, err
	}

	defer resp.Close()
	var results topTracksResults
	dec := json.NewDecoder(resp)
	if err := dec.Decode(&results); err != nil {
		return nil, fmt.Errorf("genre decode error: %w", err)
	}

	tracks := make([]data.Track, len(results.Tracks))
	for i, track := range results.Tracks {
		tracks[i] = data.Track{
			SpotifyID:  track.ID,
			Name:       track.Name,
			Popularity: track.Popularity,

			AlbumSpotifyID: track.Album.ID,
			AlbumName:      track.Album.Name,
			DiscNumber:     track.DiscNumber,
			TrackNumber:    track.TrackNumber,
			Artists:        make([]data.Artist, len(track.Artists)),
		}
		for j, artist := range track.Artists {
			tracks[i].Artists[j] = data.Artist{
				SpotifyID: artist.ID,
				Name:      artist.Name,
			}
		}
	}

	return tracks, nil
}

type topTracksResults struct {
	Tracks []struct {
		ID         string
		Name       string
		Popularity int64

		Album struct {
			ID   string
			Name string
		}
		DiscNumber  int64
		TrackNumber int64

		Artists []struct {
			ID   string
			Name string
		}
	}
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
					maxSize = image.Width
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

func (spo *Client) fetchGenrePage(ctx context.Context, name string, offset int) (*genreSearchResultsPage, error) {
	query := url.Values{}
	query.Add("query", fmt.Sprintf(`genre:"%s"`, name))
	query.Add("type", "artist")
	query.Add("limit", "50")
	query.Add("offset", fmt.Sprintf("%d", offset))

	resp, err := spo.get(ctx, "https://api.spotify.com/v1/search", query)
	if err != nil {
		return nil, err
	}

	defer resp.Close()
	var results genreSearchResultsPage
	dec := json.NewDecoder(resp)
	if err := dec.Decode(&results); err != nil {
		return nil, fmt.Errorf("genre decode error: %w", err)
	}

	return &results, nil
}

type genreSearchResultsPage struct {
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

func (spo *Client) nextReqAt() time.Time {
	return *spo.nextReqAtPtr.Load()
}

func (spo *Client) setNextReqAt(to time.Time) {
	spo.nextReqAtPtr.Store(&to)
}

func (spo *Client) get(ctx context.Context, baseURL string, query url.Values) (io.ReadCloser, error) {
	spo.mu.Lock()
	defer spo.mu.Unlock()

retry:
	nextReqAt := spo.nextReqAt()
	if !nextReqAt.IsZero() {
		now := time.Now()
		if nextReqAt.Sub(now) > time.Second {
			log.Printf("next request in %s at %s", nextReqAt.Sub(now).Truncate(time.Second), nextReqAt.Format(time.StampMilli))
		}
	wait:
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Until(nextReqAt)):
			break wait
		}
		if err := os.Remove(nextReqFilename); err != nil && !errors.Is(err, os.ErrNotExist) {
			panic(err)
		}
	}

	url, _ := url.Parse(baseURL)
	url.RawQuery = query.Encode()
	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("request error: %w", err)
	}

	token, err := spo.token()
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request error: %w", err)
	}
	if resp.StatusCode == 429 {
		spo.delay = 2 * spo.delay
		var nextReqAt time.Time
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter == "" {
			log.Printf("no retry-after header on 429; retrying in 1 minute")
			nextReqAt = time.Now().Add(time.Minute)
		} else {
			seconds, err := strconv.ParseInt(retryAfter, 10, 64)
			if err != nil {
				return nil, err
			}
			waitTime := time.Duration(seconds)*time.Second + time.Second
			log.Printf("429; retrying in %s", waitTime)
			nextReqAt = time.Now().Add(waitTime)
		}
		spo.setNextReqAt(nextReqAt)
		if err := os.WriteFile(nextReqFilename, []byte(nextReqAt.Format(time.UnixDate)), 0666); err != nil {
			return nil, err
		}
		goto retry
	}
	if err := request.Error(resp); err != nil {
		return nil, fmt.Errorf("fetch error: %w", err)
	}

	spo.setNextReqAt(time.Now().Add(spo.delay))

	return resp.Body, nil
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
