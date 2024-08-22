package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/amonks/genres/data"
	"github.com/amonks/genres/db"
	"github.com/amonks/genres/subcmd"
)

func path(ctx context.Context, db *db.DB, args []string) error {
	fs := subcmd.New("path", "create a playlist along a linear path between two tracks")
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
		"distance",
	}
	fmt.Fprintf(tw, strings.Join(header, "\t")+"\n")

	printTrack := func(track *data.Track, distance float64) {
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
			fmt.Sprintf("%f", distance),
		}, "\t")+"\n")
	}

	printTrack(fromTrack, 0)

	for i, vec := range path {
		results, err := db.NearestTracks(ctx, 1, vec)
		if err != nil {
			return fmt.Errorf("error getting nearest track for step %d: %w", i+1, err)
		}
		printTrack(&results[0], vec.Distance(results[0].Vector()))
	}

	tw.Flush()

	return nil
}
