package workers

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/amonks/genres/db"
	"github.com/amonks/genres/spotify"
	"golang.org/x/sync/errgroup"
)

type worker struct {
	f         func(context.Context, chan<- struct{}) error
	isRunning bool
}

type engine struct {
	mu      sync.Mutex
	workers map[string]worker
}

func (eng *engine) add(name string, f func(context.Context, chan<- struct{}) error) {
	eng.mu.Lock()
	defer eng.mu.Unlock()

	eng.workers[name] = worker{f: f}
}

func (eng *engine) start(ctx context.Context) error {
	ctx, cancel := context.WithCancelCause(ctx)

	g := new(errgroup.Group)
	events := make(chan string)

	run := func(name string) {
		worker := eng.workers[name]
		worker.isRunning = true
		f := worker.f
		eng.workers[name] = worker

		g.Go(func() error {
			theseEvents := make(chan struct{})
			go func() {
				for range theseEvents {
					events <- name
				}
			}()
			log.Printf("start:\t%s", name)
			if theseEvents == nil {
				return fmt.Errorf("theseEvents is nil")
			}
			if f == nil {
				return fmt.Errorf("f is nil")
			}
			err := f(ctx, theseEvents)
			if err != nil {
				log.Printf("error:\t%s\t%s", name, err)
				cancel(err)
			} else {
				log.Printf("done:\t%s", name)
			}
			return err
		})
	}

	func() {
		eng.mu.Lock()
		defer eng.mu.Unlock()

		for name := range eng.workers {
			run(name)
		}
	}()

	retrigger := func(name string) {
		eng.mu.Lock()
		defer eng.mu.Unlock()

		if eng.workers[name].isRunning {
			return
		}

		log.Printf("RETRIGGER:\t%s", name)
		run(name)
	}

	go func() {
		for ev := range events {
			log.Printf("batch:\t%s", ev)

			switch ev {

			case "genres":
				retrigger("genre_artists")

			case "track_analysis":
				retrigger("indexer")

			case "album_tracks":
				retrigger("track_analysis")

			case "artist_albums":
				retrigger("album_tracks")

			case "artist_tracks":
				retrigger("track_analysis")
			}
		}
	}()

	g.Wait()

	return nil
}

func Run(ctx context.Context, db *db.DB, spo *spotify.Client) error {
	eng := engine{
		workers: map[string]worker{},
	}

	eng.add("album_tracks", func(ctx context.Context, c chan<- struct{}) error { return runAlbumTracksFetcher(ctx, c, db, spo) })
	eng.add("artist_albums", func(ctx context.Context, c chan<- struct{}) error { return runArtistAlbumsFetcher(ctx, c, db, spo) })
	eng.add("artist_tracks", func(ctx context.Context, c chan<- struct{}) error { return runArtistTracksFetcher(ctx, c, db, spo) })
	eng.add("genre_artists", func(ctx context.Context, c chan<- struct{}) error { return runGenreArtistsFetcher(ctx, c, db, spo) })
	eng.add("genres", func(ctx context.Context, c chan<- struct{}) error { return runGenresFetcher(ctx, c, db) })
	eng.add("reporter", func(ctx context.Context, c chan<- struct{}) error { return runReporter(ctx, c, db, time.Minute*10) })
	eng.add("indexer", func(ctx context.Context, c chan<- struct{}) error { return runIndexer(ctx, c, db) })
	eng.add("track_analysis", func(ctx context.Context, c chan<- struct{}) error { return runTrackAnalysisFetcher(ctx, c, db, spo) })

	return eng.start(ctx)
}
