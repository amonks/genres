package main

import (
	"context"
	"fmt"
	"os"

	"github.com/amonks/genres/db"
	"github.com/amonks/genres/spotify"
	"github.com/amonks/genres/subcmd"
	"github.com/amonks/genres/workers"
)

func fetch(ctx context.Context, db *db.DB, args []string) error {
	subcmd := subcmd.New("fetch", "fetch data from spotify to populate the database\nrequires SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET")
	if err := subcmd.Parse(args); err != nil {
		return fmt.Errorf("flag parsing err: %w", err)
	}

	clientID, clientSecret := os.Getenv("SPOTIFY_CLIENT_ID"), os.Getenv("SPOTIFY_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		return fmt.Errorf("must set SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET")
	}
	spo := spotify.New(clientID, clientSecret)
	return workers.Run(ctx, db, spo)
}
