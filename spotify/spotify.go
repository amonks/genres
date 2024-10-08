package spotify

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/amonks/genres/data"
	"github.com/amonks/genres/limiter"
	"github.com/amonks/genres/readthrough"
	"github.com/amonks/genres/request"
)

const (
	nextReqFilename = "next-req"
	cacheDir        = "/data/tank/genres/req-cache"
	// cacheDir = "req-cache"
)

// New creates a new Spotify client, with the given clientID and clientSecret.
func New(clientID, clientSecret string) (*Client, error) {
	lim := limiter.New(nextReqFilename, time.Second)
	if err := lim.Load(); err != nil {
		return nil, err
	}

	client := &Client{
		lim:   lim,
		cache: readthrough.New(cacheDir, "req-"),

		clientID:     clientID,
		clientSecret: clientSecret,
	}

	return client, nil
}

type Client struct {
	mu sync.Mutex

	lim   *limiter.Limiter
	cache *readthrough.ReadThrough

	clientID     string
	clientSecret string

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

	albums := make([]data.Album, len(results.Albums))
	for i, fetched := range results.Albums {
		var imageURL string
		if len(fetched.Images) > 0 {
			imageURL = fetched.Images[0].URL
		}
		albums[i] = data.Album{
			SpotifyID:            fetched.ID,
			Name:                 fetched.Name,
			Type:                 fetched.AlbumType,
			ImageURL:             imageURL,
			TotalTracks:          fetched.TotalTracks,
			FetchedTracksAt:      sql.NullTime{Time: time.Now(), Valid: true},
			ReleaseDate:          fetched.ReleaseDate,
			ReleaseDatePrecision: fetched.ReleaseDatePrecision,
			Artists:              make([]data.Artist, len(fetched.Artists)),
			Tracks:               make([]data.Track, len(fetched.Tracks.Items)),
			Genres:               fetched.Genres,
			Popularity:           fetched.Popularity,
		}
		for j, artist := range fetched.Artists {
			if artist.ID == "" {
				return nil, fmt.Errorf("no id for album-artist %d", j)
			}
			albums[i].Artists[j] = data.Artist{
				SpotifyID: artist.ID,
				Name:      artist.Name,
			}
		}
		for j, track := range fetched.Tracks.Items {
			artists := make([]data.Artist, len(track.Artists))
			for k, artist := range track.Artists {
				if artist.ID == "" {
					return nil, fmt.Errorf("no id for track-artist %d on '%s'", k, track.ID)
				}
				artists[k] = data.Artist{
					SpotifyID: artist.ID,
					Name:      artist.Name,
				}
			}
			albums[i].Tracks[j] = data.Track{
				SpotifyID:  track.ID,
				Name:       track.Name,
				Popularity: track.Popularity,

				AlbumSpotifyID: fetched.ID,
				AlbumName:      fetched.Name,
				DiscNumber:     track.DiscNumber,
				TrackNumber:    track.TrackNumber,
				Artists:        artists,
			}
		}

		// Skip paginated tracks if we already got all of 'em on the
		// first page.
		if len(fetched.Tracks.Items) >= fetched.Tracks.Total {
			continue
		}

		// Fetch pages upon pages of tracks if need be.
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
					if artist.ID == "" {
						return nil, fmt.Errorf("no id for track-artist %d on '%s'", i, track.ID)
					}
					artists[i] = data.Artist{
						SpotifyID: artist.ID,
						Name:      artist.Name,
					}
				}
				albums[i].Tracks = append(albums[i].Tracks, data.Track{
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
		Genres     []string
		Popularity int64
	}
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
		for i, album := range resp.Items {
			if album.ID == "" {
				return nil, fmt.Errorf("empty spotify id for album %d", i)
			}
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
			DurationMS:    track.DurationMS,

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
		DurationMS    int64 `json:"duration_ms"`

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

var ErrSpotify = errors.New("<spotify error>")

func (spo *Client) get(ctx context.Context, baseURL string, query url.Values) (io.ReadCloser, error) {
	spo.mu.Lock()
	defer spo.mu.Unlock()

	url, _ := url.Parse(baseURL)
	url.RawQuery = query.Encode()

	if got, key, err := spo.cache.Get(url.String()); err != nil && !errors.Is(err, readthrough.ErrMiss) {
		return nil, err
	} else if err == nil {
		log.Printf("[spotify] cache hit for '%s'", key)
		return got, nil
	}

retry:
	if err := spo.lim.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

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
	switch resp.StatusCode {
	case 429:
		if err := spo.lim.SetNextAt(resp.Header.Get("Retry-After")); err != nil {
			return nil, err
		}
		goto retry

	case 502:
		spo.lim.DelayBy(time.Minute)
		goto retry
	}
	if err := request.Error(resp); err != nil {
		bs, dumpErr := httputil.DumpRequest(req, false)
		if dumpErr != nil {
			log.Printf("error dumping request: %s", dumpErr)
			return nil, errors.Join(ErrSpotify, fmt.Errorf("error fetching:\n--req--\n%s\n--error--\n%w\n^^^^^^^^^", url.String(), err))
		} else {
			return nil, errors.Join(ErrSpotify, fmt.Errorf("error fetching:\n%s\n--resp--\n%w\n^^^^^^^^", string(bs), err))
		}
	}

	spo.lim.Delay()

	r, hash, err := spo.cache.Set(url.String(), resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error writing cache file '%s': %w", hash, err)
	}

	return r, nil
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
