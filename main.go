// this program tries to populate a sqlite3 database file with genres and
// artists from spotify.
//
// see db/schema.sql for info about the resulting database.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/amonks/genres/data"
	"github.com/amonks/genres/db"
	"github.com/amonks/genres/enao"
	"github.com/amonks/genres/sigctx"
	"github.com/amonks/genres/spotify"
)

func main() {
	if err := run(); err != nil && !errors.Is(err, context.Canceled) {
		panic(err)
	} else if err != nil {
		fmt.Println("canceled")
	} else {
		fmt.Println("done")
	}
}

func run() error {
	ctx := sigctx.New()

	db, err := db.Open("genres.db")
	if err != nil {
		return err
	}
	defer func() {
		pool, err := db.DB.DB()
		if err != nil {
			panic(err)
		}
		pool.Close()
	}()

	spo := spotify.New(os.Getenv("SPOTIFY_CLIENT_ID"), os.Getenv("SPOTIFY_CLIENT_SECRET"))

	if err := populateGenres(db); err != nil {
		return err
	}
	if err := populateArtists(ctx, db, spo); err != nil {
		return err
	}

	return nil
}

func populateGenres(db *db.DB) error {
	genres, err := enao.AllGenres()
	if err != nil {
		return err
	}

	for _, genre := range genres {
		if err := db.InsertGenre(genre); err != nil {
			return err
		}
	}

	return nil
}

func populateArtists(ctx context.Context, db *db.DB, spo *spotify.Client) error {
	genres := []data.Genre{}
	if err := db.Find(&genres).Error; err != nil {
		return err
	}

	allArtists := map[string]struct{}{}

	for i, genre := range genres {
		if err := ctx.Err(); err != nil {
			return err
		}

		artists, err := spo.FetchGenre(ctx, genre.Name)
		if err != nil {
			return err
		}
		if len(artists) == 0 {
			return fmt.Errorf("no artists for genre '%s'", genre.Name)
		}

		for _, artist := range artists {
			allArtists[artist.SpotifyID] = struct{}{}
			if err := db.InsertArtist(&artist); err != nil {
				return err
			}
		}

		log.Printf("%s (%d of %d): %d artists (%d total)",
			genre.Name, i, len(genres), len(artists), len(allArtists))
	}

	return nil
}
