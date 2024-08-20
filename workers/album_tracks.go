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

		tracks, err := spo.FetchAlbumsTracks(ctx, albums)
		if err != nil {
			return err
		}
		for _, track := range tracks {
			if err := db.InsertTrack(ctx, &track); err != nil {
				return err
			}
		}
		for _, album := range albums {
			if err := db.MarkAlbumTracksFetched(album); err != nil {
				return err
			}
		}

		c <- struct{}{}
	}
}
