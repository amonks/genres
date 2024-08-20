package main

import (
	"context"
	"fmt"
	"log"

	"github.com/amonks/genres/db"
)

func index(ctx context.Context, db *db.DB) error {
	count := 0
	batchSize := 10
	total, err := db.CountTracksToIndex(ctx)
	if err != nil {
		return fmt.Errorf("error counting tracks to index: %w", err)
	}

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
		count += len(todo)
		log.Printf("indexed %d of %d (%.2f%%)", count, total, 100.0*float64(count)/float64(total))
	}
}
