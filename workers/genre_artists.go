package workers

import (
	"context"
	"fmt"

	"github.com/amonks/genres/db"
	"github.com/amonks/genres/spotify"
)

func runGenreArtistsFetcher(ctx context.Context, c chan<- struct{}, db *db.DB, spo *spotify.Client) error {
	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("canceled: %w", err)
		}

		genres, err := db.GetGenresToFetchArtists(1)
		if err != nil {
			return fmt.Errorf("error getting artistless genre: %w", err)
		}
		if len(genres) == 0 {
			return nil
		}

		genreName := genres[0]

		artists, err := spo.FetchGenre(ctx, genreName)
		if err != nil {
			return err
		}
		if len(artists) == 0 {
			return fmt.Errorf("no artists for genre '%s'", genreName)
		}

		for _, artist := range artists {
			if err := db.InsertArtist(ctx, &artist); err != nil {
				return err
			}
		}

		if err := db.MarkGenreFetched(genreName); err != nil {
			return err
		}

		c <- struct{}{}
	}
}
