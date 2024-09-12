package workers

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/amonks/genres/db"
)

func runReporter(ctx context.Context, c chan<- struct{}, db *db.DB, duration time.Duration) error {
	logfile, err := os.OpenFile("log.tsv", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	defer logfile.Close()

	var todo TODO
	var todoErr error
	todo, todoErr = gatherInfo(ctx, db)
	if todoErr != nil {
		return fmt.Errorf("reporting error: %w", todoErr)
	}

	tick := time.NewTicker(duration)

	for {
		todo, todoErr = gatherInfo(ctx, db)
		if todoErr != nil {
			return fmt.Errorf("reporting error: %w", todoErr)
		}

		fmt.Fprintf(logfile,
			"%s\t"+
				"%d\t%d\t"+
				"%d\t%d\t"+
				"%d\t%d\t"+
				"%d\t"+
				"%d\t%d\n",

			time.Now().Format(time.DateTime),
			todo.TracksKnown, todo.TracksDone,
			todo.ArtistsKnown, todo.ArtistsDone,
			todo.AlbumsKnown, todo.AlbumsDone,
			todo.TracksIndexed,
			todo.ArtistAlbumsDone, todo.ArtistTracksDone,
		)
		c <- struct{}{}

		select {
		case <-ctx.Done():
			return context.Canceled

		case <-tick.C:
		}

	}
}

type TODO struct {
	GenresToFetchArtists  int
	ArtistsToFetchTracks  int
	ArtistsToFetchAlbums  int
	AlbumsToFetchTracks   int
	TracksToFetchAnalysis int

	ArtistsKnown, ArtistsDone              int
	ArtistAlbumsDone, ArtistTracksDone     int
	AlbumsKnown, AlbumsDone                int
	TracksKnown, TracksDone, TracksIndexed int
}

func gatherInfo(ctx context.Context, db *db.DB) (TODO, error) {
	todo := TODO{}
	if count, err := db.CountTracksKnown(); err != nil {
		return todo, err
	} else {
		todo.TracksKnown = count
	}
	if err := ctx.Err(); err != nil {
		return TODO{}, fmt.Errorf("canceled: %w", err)
	}

	if count, err := db.CountTracksDone(); err != nil {
		return todo, err
	} else {
		todo.TracksDone = count
	}
	if err := ctx.Err(); err != nil {
		return TODO{}, fmt.Errorf("canceled: %w", err)
	}

	if count, err := db.CountTracksIndexed(); err != nil {
		return todo, err
	} else {
		todo.TracksIndexed = count
	}
	if err := ctx.Err(); err != nil {
		return TODO{}, fmt.Errorf("canceled: %w", err)
	}

	if count, err := db.CountAlbumsKnown(); err != nil {
		return todo, err
	} else {
		todo.AlbumsKnown = count
	}
	if err := ctx.Err(); err != nil {
		return TODO{}, fmt.Errorf("canceled: %w", err)
	}

	if count, err := db.CountAlbumsDone(); err != nil {
		return todo, err
	} else {
		todo.AlbumsDone = count
	}
	if err := ctx.Err(); err != nil {
		return TODO{}, fmt.Errorf("canceled: %w", err)
	}

	if count, err := db.CountArtistsKnown(); err != nil {
		return todo, err
	} else {
		todo.ArtistsKnown = count
	}
	if err := ctx.Err(); err != nil {
		return TODO{}, fmt.Errorf("canceled: %w", err)
	}

	if count, err := db.CountArtistAlbumsDone(); err != nil {
		return todo, err
	} else {
		todo.ArtistAlbumsDone = count
	}
	if err := ctx.Err(); err != nil {
		return TODO{}, fmt.Errorf("canceled: %w", err)
	}

	if count, err := db.CountArtistTracksDone(); err != nil {
		return todo, err
	} else {
		todo.ArtistTracksDone = count
	}
	if err := ctx.Err(); err != nil {
		return TODO{}, fmt.Errorf("canceled: %w", err)
	}

	if count, err := db.CountArtistsDone(); err != nil {
		return todo, err
	} else {
		todo.ArtistsDone = count
	}
	if err := ctx.Err(); err != nil {
		return TODO{}, fmt.Errorf("canceled: %w", err)
	}

	return todo, nil
}
