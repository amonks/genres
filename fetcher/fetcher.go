package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

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

	ArtistsKnown, ArtistsDone int
	AlbumsKnown, AlbumsDone   int
	TracksKnown, TracksDone   int
}

func (f *Fetcher) Report() (TODO, error) {
	todo := TODO{}
	if count, err := f.db.CountGenresToFetchArtists(); err != nil {
		return todo, err
	} else {
		todo.GenresToFetchArtists = count
	}
	if count, err := f.db.CountArtistsToFetchTracks(); err != nil {
		return todo, err
	} else {
		todo.ArtistsToFetchTracks = count
	}
	if count, err := f.db.CountArtistsToFetchAlbums(); err != nil {
		return todo, err
	} else {
		todo.ArtistsToFetchAlbums = count
	}
	if count, err := f.db.CountAlbumsToFetchTracks(); err != nil {
		return todo, err
	} else {
		todo.AlbumsToFetchTracks = count
	}
	if count, err := f.db.CountTracksToFetchAnalysis(); err != nil {
		return todo, err
	} else {
		todo.TracksToFetchAnalysis = count
	}
	if count, err := f.db.CountTracksKnown(); err != nil {
		return todo, err
	} else {
		todo.TracksKnown = count
	}
	if count, err := f.db.CountTracksDone(); err != nil {
		return todo, err
	} else {
		todo.TracksDone = count
	}
	if count, err := f.db.CountAlbumsKnown(); err != nil {
		return todo, err
	} else {
		todo.AlbumsKnown = count
	}
	if count, err := f.db.CountAlbumsDone(); err != nil {
		return todo, err
	} else {
		todo.AlbumsDone = count
	}
	if count, err := f.db.CountArtistsKnown(); err != nil {
		return todo, err
	} else {
		todo.ArtistsKnown = count
	}
	if count, err := f.db.CountArtistsDone(); err != nil {
		return todo, err
	} else {
		todo.ArtistsDone = count
	}

	return todo, nil
}

func (f *Fetcher) Run(ctx context.Context) error {
	logfile, err := os.OpenFile("log.tsv", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	defer logfile.Close()

	var todo TODO
	var todoErr error
	todo, todoErr = f.Report()
	if todoErr != nil {
		return fmt.Errorf("reporting error: %w", err)
	}

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		if todo.GenresToFetchArtists > 0 {
			genres, err := f.db.GetGenresToFetchArtists(1)
			if err != nil {
				return fmt.Errorf("error getting artistless genre: %w", err)
			}
			if err := f.fetchGenreArtists(ctx, genres[0]); err != nil {
				return fmt.Errorf("error fetching genre artists for '%s': %w", genres[0], err)
			}
		}

		if todo.ArtistsToFetchTracks > 0 {
			artists, err := f.db.GetArtistsToFetchTracks(1)
			if err != nil {
				return err
			}
			if err := f.fetchArtistTracks(ctx, artists[0]); err != nil {
				return err
			}
		}

		if todo.ArtistsToFetchAlbums > 0 {
			artists, err := f.db.GetArtistsToFetchAlbums(1)
			if err != nil {
				return err
			}
			if err := f.fetchArtistAlbums(ctx, artists[0]); err != nil {
				return err
			}
		}

		if todo.AlbumsToFetchTracks >= 20 {
			albums, err := f.db.GetAlbumsToFetchTracks(20)
			if err != nil {
				return err
			}
			if err := f.fetchAlbumTracks(ctx, albums); err != nil {
				return err
			}
		}

		if todo.TracksToFetchAnalysis >= 100 {
			tracks, err := f.db.GetTracksToFetchAnalysis(100)
			if err != nil {
				return err
			}
			if err := f.fetchTrackAnalyses(ctx, tracks); err != nil {
				return err
			}
		}

		todo, todoErr = f.Report()
		if todoErr != nil {
			return fmt.Errorf("reporting error: %w", err)
		}

		fmt.Fprintf(logfile, "%s\t"+
			"%d\t%d\t"+
			"%d\t%d\t"+
			"%d\t%d\n",

			time.Now().Format(time.DateTime),
			todo.TracksKnown, todo.TracksDone,
			todo.ArtistsKnown, todo.ArtistsDone,
			todo.AlbumsKnown, todo.AlbumsDone,
		)
		fmt.Println("")
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

func (f *Fetcher) fetchGenreArtists(ctx context.Context, genreName string) error {
	artists, err := f.spo.FetchGenre(ctx, genreName)
	if err != nil {
		return err
	}
	if len(artists) == 0 {
		return fmt.Errorf("no artists for genre '%s'", genreName)
	}

	for _, artist := range artists {
		if err := f.db.InsertArtist(ctx, &artist); err != nil {
			return err
		}
	}

	if err := f.db.MarkGenreFetched(genreName); err != nil {
		return err
	}

	log.Printf("fetched %d artists for genre %s", len(artists), genreName)

	return nil
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
		if err := f.db.InsertTrack(ctx, &track); err != nil {
			return err
		}
	}
	if err := f.db.MarkArtistFetched(artist); err != nil {
		return err
	}

	log.Printf("fetched %d tracks for artist %s", len(tracks), artist)

	return nil
}

func (f *Fetcher) fetchArtistAlbums(ctx context.Context, artist string) error {
	albums, err := f.spo.FetchArtistAlbums(ctx, artist)
	if err != nil {
		return err
	}
	for _, album := range albums {
		if err := f.db.InsertAlbum(ctx, &album); err != nil {
			return err
		}
	}
	if err := f.db.MarkArtistAlbumsFetched(artist); err != nil {
		return err
	}

	log.Printf("fetched %d albums for artist %s", len(albums), artist)

	return nil
}

func (f *Fetcher) fetchAlbumTracks(ctx context.Context, albums []string) error {
	tracks, err := f.spo.FetchAlbumsTracks(ctx, albums)
	if err != nil {
		return err
	}
	for _, track := range tracks {
		if err := f.db.InsertTrack(ctx, &track); err != nil {
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

func (f *Fetcher) fetchTrackAnalyses(ctx context.Context, tracks []string) error {
	analyses, err := f.spo.FetchTrackAnalyses(ctx, tracks)
	if err != nil {
		return err
	}
	if len(analyses) < len(tracks) {
		got := make(map[string]struct{}, len(analyses))
		for _, analysis := range analyses {
			got[analysis.SpotifyID] = struct{}{}
		}
		var failed []string
		for _, expected := range tracks {
			if _, ok := got[expected]; !ok {
				failed = append(failed, expected)
			}
		}
		if err := f.db.MarkTrackAnalysisFailed(failed); err != nil {
			return fmt.Errorf("error marking %d tracks as failed", len(failed))
		}
	}
	for _, track := range analyses {
		if track.SpotifyID == "" {
			bs, _ := json.MarshalIndent(analyses, "", "  ")
			panic(string(bs))
		}
		if err := f.db.AddTrackAnalysis(&track); err != nil {
			return err
		}
	}
	log.Printf("fetched %d track analyses", len(analyses))
	return nil
}
