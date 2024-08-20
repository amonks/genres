// this program tries to populate a sqlite3 database file with genres and
// artists from spotify.
//
// see db/schema.sql for info about the resulting database.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/amonks/genres/db"
	"github.com/amonks/genres/fetcher"
	"github.com/amonks/genres/sigctx"
	"github.com/amonks/genres/spotify"
)

func main() {
	if err := run(); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, flag.ErrHelp) {
		panic(err)
	}
}

var usage = strings.TrimSpace(`
usage: genres $cmd
valid $cmd are 'fetch', 'search', 'find', 'path', 'index'
for help: genres $cmd -help
`)

func run() error {
	ctx := sigctx.New()

	db, err := db.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	if len(os.Args) < 2 {
		return fmt.Errorf(usage)
	}
	cmd, args := os.Args[1], os.Args[2:]

	switch cmd {
	case "fetch":
		spo := spotify.New(os.Getenv("SPOTIFY_CLIENT_ID"), os.Getenv("SPOTIFY_CLIENT_SECRET"))
		f := fetcher.New(db, spo)
		if err := f.Run(ctx); err != nil {
			return fmt.Errorf("fetch error: %w", err)
		}

	case "index":
		return index(ctx, db)

	case "search":
		return search(ctx, db, args)

	case "find":
		return find(ctx, db, args)

	case "path":
		return path(ctx, db, args)

	default:
		return fmt.Errorf("unknown cmd: '%s'\n%s", cmd, usage)
	}

	return nil
}
