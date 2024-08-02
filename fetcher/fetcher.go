package fetcher

import (
	"context"
	"fmt"
	"log"

	"github.com/amonks/genres/db"
	"github.com/amonks/genres/enao"
	"github.com/amonks/genres/spotify"
)

type Fetcher struct {
	db  *db.DB
	spo *spotify.Client
}

func New(db *db.DB, spo *spotify.Client) *Fetcher {
	return &Fetcher{
		db:  db,
		spo: spo,
	}
}

type TODO struct {
	GenresToFetchArtists  int
	ArtistsToFetchTracks  int
	ArtistsToFetchAlbums  int
	AlbumsToFetchTracks   int
	TracksToFetchAnalysis int

	Requests int
}

func (f *Fetcher) Report() (TODO, error) {
	todo := TODO{}
	if count, err := f.countGenresToFetchArtists(); err != nil {
		return todo, err
	} else {
		todo.GenresToFetchArtists = count
	}
	if count, err := f.countArtistsToFetchTracks(); err != nil {
		return todo, err
	} else {
		todo.ArtistsToFetchTracks = count
	}
	if count, err := f.countArtistsToFetchAlbums(); err != nil {
		return todo, err
	} else {
		todo.ArtistsToFetchAlbums = count
	}
	if count, err := f.countAlbumsToFetchTracks(); err != nil {
		return todo, err
	} else {
		todo.AlbumsToFetchTracks = count
	}
	if count, err := f.countTracksToFetchAnalysis(); err != nil {
		return todo, err
	} else {
		todo.TracksToFetchAnalysis = count
	}

	todo.Requests = todo.GenresToFetchArtists +
		todo.ArtistsToFetchTracks +
		todo.ArtistsToFetchAlbums +
		todo.AlbumsToFetchTracks +
		(todo.TracksToFetchAnalysis / 100)

	return todo, nil
}

func (f *Fetcher) Run(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		todo, err := f.Report()
		if err != nil {
			return fmt.Errorf("reporting error: %w", err)
		}
		log.Printf("%d requests remain", todo.Requests)

		if todo.GenresToFetchArtists > 0 {
			genres, err := f.getGenresToFetchArtists(1)
			if err != nil {
				return fmt.Errorf("error getting artistless genre", err)
			}
			if err := f.fetchGenreArtists(ctx, genres[0]); err != nil {
				return fmt.Errorf("error fetching genre artists for '%s': %w", genres[0], err)
			}
		}

		if todo.ArtistsToFetchTracks > 0 {
			artists, err := f.getArtistsToFetchTracks(1)
			if err != nil {
				return err
			}
			if err := f.fetchArtistTracks(ctx, artists[0]); err != nil {
				return err
			}
		}

		if todo.ArtistsToFetchAlbums > 0 {
			artists, err := f.getArtistsToFetchAlbums(1)
			if err != nil {
				return err
			}
			if err := f.fetchArtistAlbums(ctx, artists[0]); err != nil {
				return err
			}
		}

		if todo.AlbumsToFetchTracks > 0 {
			albums, err := f.getAlbumsToFetchTracks(20)
			if err != nil {
				return err
			}
			if err := f.fetchAlbumTracks(ctx, albums); err != nil {
				return err
			}
		}

		if todo.TracksToFetchAnalysis > 0 {
			tracks, err := f.getTracksToFetchAnalysis(100)
			if err != nil {
				return err
			}
			if err := f.fetchTrackAnalysis(ctx, tracks); err != nil {
				return err
			}
		}
	}
}

func (f *Fetcher) populateGenres() error {
	genres, err := enao.AllGenres()
	if err != nil {
		return err
	}

	for _, genre := range genres {
		if err := f.db.InsertGenre(genre); err != nil {
			return err
		}
	}

	return nil
}

func (f *Fetcher) countGenresToFetchArtists() (int, error) {
	var count int64
	if err := f.db.
		Table("genres").
		Where("has_fetched_artists = false").
		Count(&count).
		Error; err != nil {
		return 0, fmt.Errorf("error counting genres that need to fetch artists: %w", err)
	}
	return int(count), nil
}

func (f *Fetcher) getGenresToFetchArtists(limit int) ([]string, error) {
	genreNames := []string{}
	if err := f.db.
		Table("genres").
		Limit(limit).
		Where("has_fetched_artists = false").
		Pluck("name", &genreNames).
		Error; err != nil {
		return nil, err
	}
	return genreNames, nil
}

func (f *Fetcher) fetchGenreArtists(ctx context.Context, genreName string) error {
	artists, err := f.spo.FetchGenre(ctx, genreName)
	if err != nil {
		return err
	}
	if len(artists) == 0 {
		return fmt.Errorf("no artists for genre '%s'", genreName)
	}

	for _, artist := range artists {
		if err := f.db.InsertArtist(&artist); err != nil {
			return err
		}
	}

	if err := f.db.MarkGenreFetched(genreName); err != nil {
		return err
	}

	log.Printf("fetched %d artists for genre %s", len(artists), genreName)

	return nil
}

func (f *Fetcher) countArtistsToFetchTracks() (int, error) {
	var count int64
	if err := f.db.
		Table("artists").
		Where("has_fetched_tracks = false").
		Count(&count).
		Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (f *Fetcher) getArtistsToFetchTracks(limit int) ([]string, error) {
	artists := []string{}
	if err := f.db.
		Table("artists").
		Limit(limit).
		Where("has_fetched_tracks = false").
		Pluck("spotify_id", &artists).
		Error; err != nil {
		return nil, err
	}
	return artists, nil
}

func (f *Fetcher) fetchArtistTracks(ctx context.Context, artist string) error {
	tracks, err := f.spo.FetchArtistTracks(ctx, artist)
	if err != nil {
		return err
	}
	if len(tracks) == 0 {
		log.Printf("no tracks for artist '%s'", artist)
	}
	for _, track := range tracks {
		if err := f.db.InsertTrack(&track); err != nil {
			return err
		}
	}
	if err := f.db.MarkArtistFetched(artist); err != nil {
		return err
	}

	log.Printf("fetched %d tracks for artist %s", len(tracks), artist)

	return nil
}

func (f *Fetcher) countArtistsToFetchAlbums() (int, error) {
	var count int64
	if err := f.db.
		Table("artists").
		Where("has_fetched_albums = false").
		Count(&count).
		Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (f *Fetcher) getArtistsToFetchAlbums(limit int) ([]string, error) {
	artists := []string{}
	if err := f.db.
		Table("artists").
		Limit(limit).
		Where("has_fetched_albums = false").
		Pluck("spotify_id", &artists).
		Error; err != nil {
		return nil, err
	}
	return artists, nil
}

func (f *Fetcher) fetchArtistAlbums(ctx context.Context, artist string) error {
	albums, err := f.spo.FetchArtistAlbums(ctx, artist)
	if err != nil {
		return err
	}
	for _, album := range albums {
		if err := f.db.InsertAlbum(&album); err != nil {
			return err
		}
	}
	if err := f.db.MarkArtistAlbumsFetched(artist); err != nil {
		return err
	}

	log.Printf("fetched %d albums for artist %s", len(albums), artist)

	return nil
}

func (f *Fetcher) countAlbumsToFetchTracks() (int, error) {
	var count int64
	if err := f.db.
		Table("albums").
		Where("has_fetched_tracks = false").
		Count(&count).
		Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (f *Fetcher) getAlbumsToFetchTracks(limit int) ([]string, error) {
	albums := []string{}
	if err := f.db.
		Table("albums").
		Limit(limit).
		Where("has_fetched_tracks = false").
		Pluck("spotify_id", &albums).
		Error; err != nil {
		return nil, err
	}
	return albums, nil
}

func (f *Fetcher) fetchAlbumTracks(ctx context.Context, albums []string) error {
	tracks, err := f.spo.FetchAlbumsTracks(ctx, albums)
	if err != nil {
		return err
	}
	for _, track := range tracks {
		if err := f.db.InsertTrack(&track); err != nil {
			return err
		}
	}
	for _, album := range albums {
		if err := f.db.MarkAlbumTracksFetched(album); err != nil {
			return err
		}
	}

	log.Printf("fetched %d tracks from %d albums", len(tracks), len(albums))

	return nil
}

func (f *Fetcher) countTracksToFetchAnalysis() (int, error) {
	var count int64
	if err := f.db.
		Table("tracks").
		Where("has_analysis = false").
		Count(&count).
		Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (f *Fetcher) getTracksToFetchAnalysis(limit int) ([]string, error) {
	tracks := []string{}
	if err := f.db.
		Table("tracks").
		Limit(limit).
		Where("has_analysis = false").
		Pluck("spotify_id", &tracks).
		Error; err != nil {
		return nil, err
	}
	return tracks, nil
}

func (f *Fetcher) fetchTrackAnalysis(ctx context.Context, tracks []string) error {
	analyses, err := f.spo.FetchTrackAnalyses(ctx, tracks)
	if err != nil {
		return err
	}
	if len(analyses) != len(tracks) {
		return fmt.Errorf("requested %d analyses but got %d", len(tracks), len(analyses))
	}
	for _, track := range analyses {
		if err := f.db.AddTrackAnalysis(&track); err != nil {
			return err
		}
	}
	log.Printf("fetched %d track analyses", len(analyses))
	return nil
}
