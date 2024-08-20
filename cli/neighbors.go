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

func neighbors(ctx context.Context, db *db.DB, args []string) error {
	subcmd := subcmd.New("neighbors", "return tracks similar to the given track")
	subcmd.SetArg("query", "string", "search query, matched against track, album, and artist names (required)")
	var (
		count = subcmd.Int("count", 5, "number of tracks to return")
	)
	if err := subcmd.Parse(args); err != nil {
		return fmt.Errorf("flag parsing err: %w", err)
	}

	query := strings.Join(subcmd.Args(), " ")

	results, err := db.Search(ctx, query, 1)
	if err != nil {
		return fmt.Errorf("error in search for '%s': %w", query, err)
	}

	track := results[0]
	target := track.Vector()

	tracks, err := db.NearestTracks(ctx, *count+1, target)
	if err != nil {
		return fmt.Errorf("error finding neighbors of '%s': %w", track.SpotifyID, err)
	}

	distances := make([]float64, len(tracks))
	for i, track := range tracks {
		distances[i] = target.Distance(track.Vector())
	}

	rows := make([][]string, len(tracks)+1)
	rows[0] = []string{
		"artists",
		"album", "track", "spotify_id",
		"acousticness",
		"danceability",
		"energy",
		"instrumentalness",
		"liveness",
		"speechiness",
		"valence",
		"distance",
	}
	for i, track := range tracks {
		artists := make([]string, len(track.Artists))
		for i, artist := range track.Artists {
			artists[i] = artist.Name
		}
		rows[i+1] = []string{
			strings.Join(artists, ", "),
			track.AlbumName, track.Name, track.SpotifyID,
			fmt.Sprintf("%f", track.Acousticness),
			fmt.Sprintf("%f", track.Danceability),
			fmt.Sprintf("%f", track.Energy),
			fmt.Sprintf("%f", track.Instrumentalness),
			fmt.Sprintf("%f", track.Liveness),
			fmt.Sprintf("%f", track.Speechiness),
			fmt.Sprintf("%f", track.Valence),
			fmt.Sprintf("%f", distances[i]),
		}
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, row := range rows {
		fmt.Fprintf(tw, strings.Join(row, "\t")+"\n")
	}
	tw.Flush()

	return nil
}
