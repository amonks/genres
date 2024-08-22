package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/amonks/genres/db"
	"github.com/amonks/genres/subcmd"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func progress(ctx context.Context, db *db.DB, args []string) error {
	subcmd := subcmd.New("progress", "report progress from the fetcher")
	if err := subcmd.Parse(args); err != nil {
		return fmt.Errorf("flag parsing err: %w", err)
	}

	genresKnown, err := db.CountGenresKnown()
	if err != nil {
		return err
	}
	genresFetchedArtists, err := db.CountGenresWithFetchedArtists()
	if err != nil {
		return err
	}

	artistsKnown, err := db.CountArtistsKnown()
	if err != nil {
		return err
	}
	artistsFetchedAlbums, err := db.CountArtistsWithFetchedAlbums()
	if err != nil {
		return err
	}
	artistsFetchedTracks, err := db.CountArtistsWithFetchedTracks()
	if err != nil {
		return err
	}

	albumsKnown, err := db.CountAlbumsKnown()
	if err != nil {
		return err
	}
	albumsFetchedTracks, err := db.CountAlbumsWithFetchedTracks()
	if err != nil {
		return err
	}

	tracksKnown, err := db.CountTracksKnown()
	if err != nil {
		return err
	}
	tracksIndexed, err := db.CountTracksIndexed()
	if err != nil {
		return err
	}
	tracksFetchedAnalysis, err := db.CountTracksWithFetchedAnalysis()
	if err != nil {
		return err
	}

	printSection("genres", genresKnown, 6_291, map[string]int{
		"fetched artists": genresFetchedArtists,
	})
	printSection("artists", artistsKnown, 11_000_000, map[string]int{
		"fetched top tracks": artistsFetchedTracks,
		"fetched albums":     artistsFetchedAlbums,
	})
	printSection("albums", albumsKnown, 0, map[string]int{
		"fetched tracks": albumsFetchedTracks,
	})
	printSection("tracks", tracksKnown, 100_000_000, map[string]int{
		"indexed":          tracksIndexed,
		"fetched analysis": tracksFetchedAnalysis,
	})

	return nil
}

var humanPrinter = message.NewPrinter(language.English)

func printSection(name string, known, target int, done map[string]int) {
	humanPrinter.Printf("%s\n", strings.ToUpper(name))
	if target != 0 {
		humanPrinter.Printf("  %d\tknown (%.2f%%)\n", known, 100.0*float64(known)/float64(target))
	} else {
		humanPrinter.Printf("  %d\tknown\n", known)
	}
	for k, v := range done {
		humanPrinter.Printf("  %d\t%s (%.2f%%)\n", v, k, 100.0*float64(v)/float64(known))
	}
	humanPrinter.Printf("\n")
}
