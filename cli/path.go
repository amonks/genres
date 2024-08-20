package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/amonks/genres/data"
	"github.com/amonks/genres/db"
)

func path(ctx context.Context, db *db.DB, args []string) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	var (
		from  = fs.String("from", "", "query for 'from' track")
		to    = fs.String("to", "", "query for 'to' track")
		steps = fs.Int("steps", 5, "number of steps on the path")
	)
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("flag parsing err: %w", err)
	}

	fromTrack, err := db.Resolve(ctx, *from)
	if err != nil {
		return fmt.Errorf("error getting 'from' track '%s': %w", *from, err)
	}

	toTrack, err := db.Resolve(ctx, *to)
	if err != nil {
		return fmt.Errorf("error getting 'to' track '%s': %w", *to, err)
	}

	fromVec, toVec := fromTrack.Vector(), toTrack.Vector()
	delta := fromVec.Delta(toVec)
	path := fromVec.Path(delta, *steps)
	tracks := make([]data.Track, *steps+1)
	distances := make([]float64, *steps+1)

	tracks[0], distances[0] = *fromTrack, 0
	for i, vec := range path {
		results, err := db.NearestTracks(ctx, 1, vec)
		if err != nil {
			return fmt.Errorf("error getting nearest track for step %d: %w", i+1, err)
		}
		tracks[i+1] = results[0]
		distances[i+1] = vec.Distance(results[0].Vector())
	}

	rows := make([][]string, len(tracks)+1)
	rows[0] = []string{"artist", "album", "track", "spotify_id", "acousticness", "danceability", "energy", "instrumentalness", "liveness", "speechiness", "valence", "distance"}
	for i, track := range tracks {
		rows[i+1] = []string{track.Artists[0].Name, track.AlbumName, track.Name, track.SpotifyID,
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

