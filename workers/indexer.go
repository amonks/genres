package workers

import (
	"context"
	"fmt"

	"github.com/amonks/genres/db"
)

func runIndexer(ctx context.Context, c chan<- struct{}, db *db.DB, batchSize int) error {
	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("canceled: %w", err)
		}

		todo, err := db.GetTracksToIndex(ctx, batchSize)
		if err != nil {
			return fmt.Errorf("error getting %d tracks to index: %w", batchSize, err)
		}
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("canceled: %w", err)
		}
		if len(todo) == 0 {
			return nil
		}

		if err := db.IndexTracks(ctx, todo); err != nil {
			return fmt.Errorf("error indexing %d tracks: %w", len(todo), err)
		}

		c <- struct{}{}
	}
}
