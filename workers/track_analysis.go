package workers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/amonks/genres/db"
	"github.com/amonks/genres/spotify"
)

func runTrackAnalysisFetcher(ctx context.Context, c chan<- struct{}, db *db.DB, spo *spotify.Client) error {
	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("canceled: %w", err)
		}

		tracks, err := db.GetTracksToFetchAnalysis(100)
		if err != nil {
			return err
		}
		if len(tracks) == 0 {
			return nil
		}

		analyses, err := spo.FetchTrackAnalyses(ctx, tracks)
		if err != nil {
			return err
		}

		if len(analyses) < len(tracks) {
			got := make(map[string]struct{}, len(analyses))
			for _, analysis := range analyses {
				got[analysis.SpotifyID] = struct{}{}
			}
			var failed []string
			for _, expected := range tracks {
				if _, ok := got[expected]; !ok {
					failed = append(failed, expected)
				}
			}
			if err := db.MarkTrackAnalysisFailed(failed); err != nil {
				return fmt.Errorf("error marking %d tracks as failed", len(failed))
			}
		}

		for _, track := range analyses {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("canceled: %w", err)
			}
			if track.SpotifyID == "" {
				bs, _ := json.MarshalIndent(analyses, "", "  ")
				panic(string(bs))
			}
			if err := db.AddTrackAnalysis(&track); err != nil {
				return err
			}
		}

		c <- struct{}{}
	}
}
