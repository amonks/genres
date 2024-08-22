package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/amonks/genres/db"
	"github.com/amonks/genres/subcmd"
)

func search(ctx context.Context, db *db.DB, args []string) error {
	subcmd := subcmd.New("search", "search the database for a track")
	subcmd.SetArg("query", "string", "search query, matched against track, album, and artist names (required)")
	count := subcmd.Int("count", 1, "number of tracks to return")
	if err := subcmd.Parse(args); err != nil {
		return fmt.Errorf("flag parsing err: %w", err)
	}

	query := strings.Join(subcmd.Args(), " ")

	tracks, err := db.Search(ctx, query, *count)
	if err != nil {
		return fmt.Errorf("error in search for '%s': %w", query, err)
	}

	if len(tracks) == 0 {
		fmt.Printf("no results for '%s'\n", query)
		return nil
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	header := []string{
		"artists",
		"album", "track", "spotify_id",
		"acousticness",
		"danceability",
		"energy",
		"instrumentalness",
		"liveness",
		"speechiness",
		"valence",
	}
	fmt.Fprintf(tw, strings.Join(header, "\t")+"\n")

	for _, track := range tracks {
		artists := make([]string, len(track.Artists))
		for i, artist := range track.Artists {
			artists[i] = artist.Name
		}
		fmt.Fprintf(tw, strings.Join([]string{
			strings.Join(artists, ", "),
			track.AlbumName, track.Name, track.SpotifyID,
			fmt.Sprintf("%f", track.Acousticness),
			fmt.Sprintf("%f", track.Danceability),
			fmt.Sprintf("%f", track.Energy),
			fmt.Sprintf("%f", track.Instrumentalness),
			fmt.Sprintf("%f", track.Liveness),
			fmt.Sprintf("%f", track.Speechiness),
			fmt.Sprintf("%f", track.Valence),
		}, "\t")+"\n")
	}

	tw.Flush()

	return nil
}
