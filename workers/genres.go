package workers

import (
	"context"
	"fmt"

	"github.com/amonks/genres/db"
	"github.com/amonks/genres/enao"
)

func runGenresFetcher(ctx context.Context, c chan<- struct{}, db *db.DB) error {
	genres, err := enao.AllGenres()
	if err != nil {
		return err
	}

	if len(genres) == 0 {
		return nil
	}

	for _, genre := range genres {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("canceled: %w", err)
		}

		if err := db.InsertGenre(genre); err != nil {
			return err
		}
	}

	c <- struct{}{}

	return nil
}
