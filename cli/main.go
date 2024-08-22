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

	"github.com/amonks/genres/db"
	"github.com/amonks/genres/sigctx"
)

func main() {
	if err := run(); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, flag.ErrHelp) {
		panic(err)
	}
}

func run() error {
	ctx := sigctx.New()

	flag.CommandLine.Init("genres", flag.ContinueOnError)
	dbFilename := flag.String("dbfile", "genres.db", "path to database file")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "\nmanipulate data from spotify\n\n")
		fmt.Fprintf(os.Stderr, "  genres [flags] $cmd <defined by cmd...>\n\n")
		fmt.Fprintf(os.Stderr, "flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "  $cmd {fetch, search, find, path, neighbors, serve, progress}\n")
		fmt.Fprintf(os.Stderr, "  \twhich command to run\n")
		fmt.Fprintf(os.Stderr, "  \trun `flag $cmd -help for more details about specific commands.\n")
	}
	if err := flag.CommandLine.Parse(os.Args[1:]); err != nil && !errors.Is(err, flag.ErrHelp) {
		return fmt.Errorf("flag parse error: %w", err)
	} else if err != nil {
		return nil
	}

	db, err := db.Open(*dbFilename)
	if err != nil {
		return fmt.Errorf("db open error: %w", err)
	}
	defer db.Close()

	args := flag.Args()
	if len(args) < 1 {
		flag.CommandLine.Parse([]string{"-help"})
		return nil
	}
	cmd, args := args[0], args[1:]
	switch cmd {
	case "serve":
		return serve(ctx, db, args)
	case "progress":
		return progress(ctx, db, args)
	case "fetch":
		return fetch(ctx, db, args)
	case "search":
		return search(ctx, db, args)
	case "find":
		return find(ctx, db, args)
	case "path":
		return path(ctx, db, args)
	case "neighbors":
		return neighbors(ctx, db, args)
	case "migrate":
		return nil
	default:
		return fmt.Errorf("unknown cmd: '%s'", cmd)
	}
}
