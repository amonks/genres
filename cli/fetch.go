package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/amonks/genres/db"
	"github.com/amonks/genres/setflag"
	"github.com/amonks/genres/spotify"
	"github.com/amonks/genres/subcmd"
	"github.com/amonks/genres/workers"
)

func fetch(ctx context.Context, db *db.DB, args []string) error {
	var allowedWorkers = []string{
		"album_tracks",
		"artist_albums",
		"genre_artists",
		"genres",
		"track_analysis",
		"album_tracks_refetch",
	}

	subcmd := subcmd.New("fetch", "fetch data from spotify to populate the database\nrequires SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET")
	fWorkers := setflag.New(allowedWorkers...)
	subcmd.Var(fWorkers, "workers", fmt.Sprintf("Workers to run; valid options are {%s}", strings.Join(allowedWorkers, ", ")))
	if err := subcmd.Parse(args); err != nil {
		return fmt.Errorf("flag parsing err: %w", err)
	}

	workersList := fWorkers.List()
	if len(workersList) == 0 {
		workersList = allowedWorkers
	}

	clientID, clientSecret := os.Getenv("SPOTIFY_CLIENT_ID"), os.Getenv("SPOTIFY_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		return fmt.Errorf("must set SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET")
	}
	spo, err := spotify.New(clientID, clientSecret)
	if err != nil {
		return fmt.Errorf("error creating spotify client: %w", err)
	}

	return workers.Run(ctx, db, spo, workersList)
}
