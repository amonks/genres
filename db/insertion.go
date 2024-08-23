package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/amonks/genres/data"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// InsertGenre, given a Genre, inserts it into the genres table.
func (db *DB) InsertGenre(genre *data.Genre) error {
	defer db.hold()()

	if genre.Name == "" {
		return fmt.Errorf("no genre name")
	}

	if err := db.rw.
		Table("genres").
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&genre).
		Error; err != nil {
		return fmt.Errorf("error inserting genre '%s': %w", genre.Name, err)
	}
	return nil
}

func (db *DB) MarkArtistAlbumsFetched(artistSpotifyID string) error {
	defer db.hold()()

	if artistSpotifyID == "" {
		return fmt.Errorf("no spotify id")
	}

	if err := db.rw.
		Table("artists").
		Where("spotify_id = ?", artistSpotifyID).
		Update("fetched_albums_at", sql.NullTime{Time: time.Now(), Valid: true}).
		Error; err != nil {
		return fmt.Errorf("error marking artist '%s' albums as fetched: %w", artistSpotifyID, err)
	}
	return nil
}

func (db *DB) MarkGenreFetched(genreName string) error {
	defer db.hold()()

	if genreName == "" {
		return fmt.Errorf("no spotify id")
	}

	if err := db.rw.
		Table("genres").
		Where("name = ?", genreName).
		Update("fetched_artists_at", sql.NullTime{Time: time.Now(), Valid: true}).
		Error; err != nil {
		return fmt.Errorf("error marking genre '%s' as fetched: %w", genreName, err)
	}
	return nil
}

func (db *DB) MarkArtistFetched(artistSpotifyID string) error {
	defer db.hold()()

	if artistSpotifyID == "" {
		return fmt.Errorf("no spotify id")
	}

	if err := db.rw.
		Table("artists").
		Where("spotify_id = ?", artistSpotifyID).
		Update("fetched_tracks_at", sql.NullTime{Time: time.Now(), Valid: true}).
		Error; err != nil {
		return fmt.Errorf("error marking artist '%s' as fetched: %w", artistSpotifyID, err)
	}
	return nil
}

func (db *DB) MarkTrackAnalysisFailed(tracks []string) error {
	defer db.hold()()

	if err := db.rw.
		Table("tracks").
		Where("spotify_id in ?", tracks).
		Update("failed_analysis_at", sql.NullTime{Time: time.Now(), Valid: true}).
		Error; err != nil {
		return fmt.Errorf("error marking %d tracks as failed", len(tracks))
	}
	return nil
}

func (db *DB) AddTrackAnalysis(track *data.Track) error {
	defer db.hold()()

	if track.SpotifyID == "" {
		return fmt.Errorf("no spotify id")
	}
	if err := db.rw.
		Table("tracks").
		Where("spotify_id = ?", track.SpotifyID).
		Updates(map[string]interface{}{
			"fetched_analysis_at": sql.NullTime{Time: time.Now(), Valid: true},

			"key":            track.Key,
			"mode":           track.Mode,
			"tempo":          track.Tempo,
			"time_signature": track.TimeSignature,

			"acousticness":     track.Acousticness,
			"danceability":     track.Danceability,
			"energy":           track.Energy,
			"instrumentalness": track.Instrumentalness,
			"liveness":         track.Liveness,
			"loudness":         track.Loudness,
			"speechiness":      track.Speechiness,
			"valence":          track.Valence,
		}).
		Error; err != nil {
		return fmt.Errorf("error adding track analysis for '%s': %w", track.SpotifyID, err)
	}

	return nil
}

func (db *DB) PopulateAlbum(ctx context.Context, album *data.Album) error {
	defer db.hold()()

	if album.SpotifyID == "" {
		return fmt.Errorf("no spotify id")
	}

	return db.rw.Transaction(func(db *gorm.DB) error {
		if err := db.
			Table("albums").
			Where("spotify_id = ?", album.SpotifyID).
			Updates(map[string]any{
				"fetched_tracks_at":      sql.NullTime{Time: time.Now(), Valid: true},
				"name":                   album.Name,
				"type":                   album.Type,
				"image_url":              album.ImageURL,
				"total_tracks":           album.TotalTracks,
				"release_date":           album.ReleaseDate,
				"release_date_precision": album.ReleaseDatePrecision,
			}).Error; err != nil {
			return fmt.Errorf("error populating album '%s': %w", album.SpotifyID, err)
		}
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("canceled: %w", err)
		}

		for _, artist := range album.Artists {
			if artist.SpotifyID == "" {
				return fmt.Errorf("no spotify id")
			}

			if err := db.
				Table("artists").
				Clauses(clause.OnConflict{DoNothing: true}).
				Create(artist).
				Error; err != nil {
				return fmt.Errorf("error inserting artist '%s' for populating album '%s': %w", artist.SpotifyID, album.SpotifyID, err)
			}
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("canceled: %w", err)
			}

			if err := db.
				Table("album_artists").
				Clauses(clause.OnConflict{DoNothing: true}).
				Create(&data.AlbumArtist{
					AlbumSpotifyID:  album.SpotifyID,
					ArtistSpotifyID: artist.SpotifyID,
				}).
				Error; err != nil {
				return fmt.Errorf("error inserting album_artist '%s' for populating album '%s': %w", artist.SpotifyID, album.SpotifyID, err)
			}
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("canceled: %w", err)
			}
		}

		for _, track := range album.Tracks {
			if err := db.
				Table("tracks").
				Clauses(clause.OnConflict{DoNothing: true}).
				Create(track).
				Error; err != nil {
				return fmt.Errorf("error inserting track '%s' from album '%s': %w", track.SpotifyID, album.SpotifyID, err)
			}
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("canceled: %w", err)
			}

			if err := db.
				Table("album_tracks").
				Clauses(clause.OnConflict{DoNothing: true}).
				Create(&data.AlbumTrack{
					AlbumSpotifyID: album.SpotifyID,
					TrackSpotifyID: track.SpotifyID,
				}).
				Error; err != nil {
				return fmt.Errorf("error inserting album_track {'%s' '%s'}: %w", album.SpotifyID, track.SpotifyID, err)
			}
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("canceled: %w", err)
			}
		}

		return nil
	})
}

func (db *DB) InsertAlbum(ctx context.Context, album *data.Album) error {
	defer db.hold()()

	if album.SpotifyID == "" {
		return fmt.Errorf("no spotify id")
	}

	return db.rw.Transaction(func(db *gorm.DB) error {
		if err := db.
			Table("albums").
			Clauses(clause.OnConflict{UpdateAll: true}).
			Create(album).
			Error; err != nil {
			return fmt.Errorf("error inserting album '%s': %w", album.SpotifyID, err)
		}
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("canceled: %w", err)
		}

		for _, artist := range album.Artists {
			if artist.SpotifyID == "" {
				return fmt.Errorf("no spotify id")
			}

			if err := db.
				Table("artists").
				Clauses(clause.OnConflict{DoNothing: true}).
				Create(artist).
				Error; err != nil {
				return fmt.Errorf("error inserting artist '%s' for album '%s': %w", artist.SpotifyID, album.SpotifyID, err)
			}
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("canceled: %w", err)
			}

			if err := db.
				Table("album_artists").
				Clauses(clause.OnConflict{DoNothing: true}).
				Create(&data.AlbumArtist{
					AlbumSpotifyID:  album.SpotifyID,
					ArtistSpotifyID: artist.SpotifyID,
				}).
				Error; err != nil {
				return fmt.Errorf("error inserting album_artist '%s' for album '%s': %w", artist.SpotifyID, album.SpotifyID, err)
			}
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("canceled: %w", err)
			}
		}

		return nil
	})
}

func (db *DB) InsertTrack(ctx context.Context, track *data.Track) error {
	defer db.hold()()

	if track.SpotifyID == "" {
		return fmt.Errorf("no spotify id")
	}

	return db.rw.Transaction(func(db *gorm.DB) error {
		if err := db.
			Table("tracks").
			Clauses(clause.OnConflict{DoNothing: true}).
			Create(track).
			Error; err != nil {
			return fmt.Errorf("error inserting track '%s': %w", track.Name, err)
		}
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("canceled: %w", err)
		}

		if track.AlbumSpotifyID != "" {
			if err := db.
				Table("albums").
				Clauses(clause.OnConflict{DoNothing: true}).
				Create(&data.Album{
					SpotifyID: track.AlbumSpotifyID,
					Name:      track.AlbumName,
				}).
				Error; err != nil {
				return fmt.Errorf("error inserting album '%s' for track '%s': %w", track.AlbumSpotifyID, track.SpotifyID, err)
			}
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("canceled: %w", err)
			}

			if err := db.
				Table("album_tracks").
				Clauses(clause.OnConflict{DoNothing: true}).
				Create(&data.AlbumTrack{
					TrackSpotifyID: track.SpotifyID,
					AlbumSpotifyID: track.AlbumSpotifyID,
				}).
				Error; err != nil {
				return fmt.Errorf("error inserting album_track '%s' for track '%s': %w", track.AlbumSpotifyID, track.SpotifyID, err)
			}
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("canceled: %w", err)
			}
		}

		for _, artist := range track.Artists {
			if artist.SpotifyID == "" {
				return fmt.Errorf("no spotify id")
			}

			if err := db.
				Table("artists").
				Clauses(clause.OnConflict{DoNothing: true}).
				Create(artist).
				Error; err != nil {
				return fmt.Errorf("error inserting artist '%s' for track '%s': %w", artist.SpotifyID, track.SpotifyID, err)
			}
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("canceled: %w", err)
			}

			if err := db.
				Table("track_artists").
				Clauses(clause.OnConflict{DoNothing: true}).
				Create(&data.TrackArtist{
					TrackSpotifyID:  track.SpotifyID,
					ArtistSpotifyID: artist.SpotifyID,
				}).
				Error; err != nil {
				return fmt.Errorf("error inserting track_artist '%s' for track '%s': %w", artist.SpotifyID, track.SpotifyID, err)
			}
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("canceled: %w", err)
			}
		}

		return nil
	})
}

// InsertArtist, given an Artist, inserts the appropriate data into the artists
// and artist_genres tables, doing nothing if they already exist.
func (db *DB) InsertArtist(ctx context.Context, artist *data.Artist) error {
	defer db.hold()()

	if artist.SpotifyID == "" {
		return fmt.Errorf("no spotify id")
	}

	return db.rw.Transaction(func(db *gorm.DB) error {
		if err := db.
			Table("artists").
			Clauses(clause.OnConflict{DoNothing: true}).
			Create(artist).
			Error; err != nil {
			return fmt.Errorf("error inserting artist '%s': %w", artist.SpotifyID, err)
		}
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("canceled: %w", err)
		}

		for _, genre := range artist.Genres {
			if genre == "" {
				return fmt.Errorf("no genre name")
			}

			if err := db.
				Table("genres").
				Clauses(clause.OnConflict{DoNothing: true}).
				Create(&data.Genre{
					Name: genre,
				}).
				Error; err != nil {
				fmt.Errorf("error inserting genre '%s' for artist '%s': %w", genre, artist.SpotifyID, err)
			}

			if err := db.
				Table("artist_genres").
				Clauses(clause.OnConflict{DoNothing: true}).
				Create(&data.ArtistGenre{
					ArtistSpotifyID: artist.SpotifyID,
					GenreName:       genre,
				}).
				Error; err != nil {
				return fmt.Errorf("error inserting artist_genre '%s' for artist '%s': %w", genre, artist.SpotifyID, err)
			}
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("canceled: %w", err)
			}

		}

		return nil
	})
}
