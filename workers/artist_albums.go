package workers

import (
	"context"
	"fmt"

	"github.com/amonks/genres/db"
	"github.com/amonks/genres/spotify"
)

func runArtistAlbumsFetcher(ctx context.Context, c chan<- struct{}, db *db.DB, spo *spotify.Client) error {
	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("canceled: %w", err)
		}

		artists, err := db.GetArtistsToFetchAlbums(1)
		if err != nil {
			return err
		}
		if len(artists) == 0 {
			return nil
		}

		artist := artists[0]

		albums, err := spo.FetchArtistAlbums(ctx, artist)
		if err != nil {
			return err
		}
		for _, album := range albums {
			if err := db.InsertAlbum(ctx, &album); err != nil {
				return err
			}
		}
		if err := db.MarkArtistAlbumsFetched(artist); err != nil {
			return err
		}

		c <- struct{}{}
	}
}
