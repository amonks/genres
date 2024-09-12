package workers

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/amonks/genres/data"
	"github.com/amonks/genres/db"
	"github.com/amonks/genres/spotify"
)

func runArtistTracksFetcher(ctx context.Context, c chan<- struct{}, db *db.DB, spo *spotify.Client) error {
	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("canceled: %w", err)
		}

		artists, err := db.GetArtistsToFetchTracks(1)
		if err != nil {
			return err
		}
		if len(artists) == 0 {
			return nil
		}

		artist := artists[0]

		tracks, err := spo.FetchArtistTracks(ctx, artist)
		if err != nil && errors.Is(err, spotify.ErrSpotify) {
			if markErr := db.MarkArtistFailed(artist); markErr != nil {
				return markErr
			}
			log.Printf("failed to fetch artist '%s': %s", artist, err)
			return nil
		} else if err != nil {
			return err
		}
		if len(tracks) == 0 {
			log.Printf("[artist_tracks] no tracks for artist '%s'", artist)
		}
		for _, track := range tracks {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("canceled: %w", err)
			}

			if err := db.InsertTrack(ctx, &track); err != nil {
				return err
			}

			if err := db.InsertAlbum(ctx, &data.Album{
				SpotifyID: track.AlbumSpotifyID,
				Name:      track.AlbumName,
			}); err != nil {
				return err
			}
		}
		if err := db.MarkArtistFetched(artist); err != nil {
			return err
		}

		c <- struct{}{}
	}
}
