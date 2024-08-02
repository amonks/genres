// this program tries to populate a sqlite3 database file with genres and
// artists from spotify.
//
// see db/schema.sql for info about the resulting database.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/amonks/genres/db"
	"github.com/amonks/genres/fetcher"
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

	f := fetcher.New(db, spo)

	flag.Parse()
	operation := "fetch"
	if args := flag.Args(); len(args) >= 1 {
		operation = args[0]
	}
	switch operation {

	case "enumerate-work":
		todo, err := f.Report()
		if err != nil {
			return fmt.Errorf("error building fetcher report: %w", err)
		}
		if json, err := json.Marshal(todo); err != nil {
			return fmt.Errorf("error marshaling fetcher report: %w", err)
		} else {
			log.Println(string(json))
		}

	case "fetch":
		if err := f.Run(ctx); err != nil {
			return fmt.Errorf("fetch error: %w", err)
		}

	case "query":
		// TODO

	default:
		return fmt.Errorf("unknown operation: '%s'", operation)
	}

	return nil
}
