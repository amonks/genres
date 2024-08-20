package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"

	"github.com/amonks/genres/db"
)

func find(ctx context.Context, db *db.DB, args []string) error {
	fs := flag.NewFlagSet("find", flag.ContinueOnError)
	var (
		count = fs.Int("count", 1, "number of tracks to return")

		acousticness     = fs.Float64("acousticness", -1, "acousticness")
		danceability     = fs.Float64("danceability", -1, "danceability")
		energy           = fs.Float64("energy", -1, "energy")
		instrumentalness = fs.Float64("instrumentalness", -1, "instrumentalness")
		liveness         = fs.Float64("liveness", -1, "liveness")
		speechiness      = fs.Float64("speechiness", -1, "speechiness")
		valence          = fs.Float64("valence", -1, "valence")
	)
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("flag parsing err: %w", err)
	}

	input := make(map[string]float64)
	if *acousticness >= 0 {
		input["acousticness"] = *acousticness
	}
	if *danceability >= 0 {
		input["danceability"] = *danceability
	}
	if *energy >= 0 {
		input["energy"] = *energy
	}
	if *instrumentalness >= 0 {
		input["instrumentalness"] = *instrumentalness
	}
	if *liveness >= 0 {
		input["liveness"] = *liveness
	}
	if *speechiness >= 0 {
		input["speechiness"] = *speechiness
	}
	if *valence >= 0 {
		input["valence"] = *valence
	}

	fmt.Println("input", input)

	tracks, err := db.NearestTracks(ctx, *count, input)
	if err != nil {
		return err
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
