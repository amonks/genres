package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/amonks/genres/db"
)

func search(ctx context.Context, db *db.DB, args []string) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	var (
		count = fs.Int("count", 1, "number of tracks to return")
	)
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("flag parsing err: %w", err)
	}

	query := strings.Join(fs.Args(), " ")

	tracks, err := db.Search(ctx, query, *count)
	if err != nil {
		return fmt.Errorf("error in search for '%s': %w", query, err)
	}

	if len(tracks) == 0 {
		fmt.Printf("no results for '%s'\n", query)
		return nil
	}

	var out interface{} = tracks
	if *count == 1 {
		out = tracks[0]
	}

	json, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(json))

	return nil
}
