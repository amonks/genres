package workers

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/amonks/genres/db"
	"github.com/amonks/genres/spotify"
)

func runAlbumTracksRefetcher(ctx context.Context, c chan<- struct{}, db *db.DB, spo *spotify.Client) error {
	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("canceled: %w", err)
		}

		const count = 20
		albums, err := db.GetAlbumsToRefetchTracks(count)
		if err != nil {
			return err
		}
		if len(albums) == 0 {
			return nil
		}
		if len(albums) < count {
			return nil
		}

		fetched, err := spo.FetchAlbums(ctx, albums)
		if err != nil && errors.Is(err, spotify.ErrSpotify) {
			if markErr := db.MarkAlbumsTracksFailed(albums); markErr != nil {
				return markErr
			}
			log.Printf("failed to fetch %d albums: %s", len(albums), err)
			return nil
		} else if err != nil {
			return err
		}
		if len(fetched) != len(albums) {
			return fmt.Errorf("expected %d albums but got %d", len(albums), len(fetched))
		}

		if err := db.PopulateAlbums(ctx, fetched); err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("canceled: %w", err)
		}

		c <- struct{}{}
	}
}
