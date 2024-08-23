package workers

import (
	"context"
	"fmt"

	"github.com/amonks/genres/db"
	"github.com/amonks/genres/spotify"
)

func runAlbumTracksFetcher(ctx context.Context, c chan<- struct{}, db *db.DB, spo *spotify.Client) error {
	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("canceled: %w", err)
		}

		albums, err := db.GetAlbumsToFetchTracks(20)
		if err != nil {
			return err
		}
		if len(albums) == 0 {
			return nil
		}
		if len(albums) < 20 {
			return nil
		}

		fetched, err := spo.FetchAlbums(ctx, albums)
		if err != nil {
			return err
		}
		if len(fetched) != len(albums) {
			return fmt.Errorf("expected %d albums but got %d", len(albums), len(fetched))
		}

		for _, album := range fetched {
			if err := db.PopulateAlbum(ctx, &album); err != nil {
				return err
			}
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("canceled: %w", err)
			}
		}

		c <- struct{}{}
	}
}
