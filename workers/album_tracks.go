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
		for _, album := range fetched {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("canceled: %w", err)
			}
			if err := db.PopulateAlbum(ctx, &album); err != nil {
				return err
			}
			for _, track := range album.Tracks {
				if err := ctx.Err(); err != nil {
					return fmt.Errorf("canceled: %w", err)
				}

				if err := db.InsertTrack(ctx, &track); err != nil {
					return err
				}
			}
		}

		c <- struct{}{}
	}
}
