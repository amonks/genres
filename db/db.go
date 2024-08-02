package db

import (
	_ "embed"
	"fmt"

	"github.com/amonks/genres/data"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DB represents our sqlite3 database file.
type DB struct{ *gorm.DB }

//go:embed schema.sql
var schema string

// Open returns a connection to a migrated sqlite3 database file on disk,
// creating the file and running migrations if necessary.
func Open(filename string) (*DB, error) {
	gdb, err := gorm.Open(sqlite.Open(filename), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("error opening db file at '%s': %w", filename, err)
	}

	db := &DB{gdb}

	if err := db.Exec(schema).Error; err != nil {
		return nil, fmt.Errorf("error migrating db at '%s': %w", err)
	}

	return db, nil
}

// InsertGenre, given a Genre, inserts it into the genres table.
func (db *DB) InsertGenre(genre *data.Genre) error {
	if err := db.
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&genre).
		Error; err != nil {
		return fmt.Errorf("error inserting genre '%s': %w", genre.Name, err)
	}
	return nil
}

func (db *DB) MarkAlbumTracksFetched(albumSpotifyID string) error {
	if err := db.
		Table("albums").
		Where("spotify_id = ?", albumSpotifyID).
		Update("has_fetched_tracks", true).
		Error; err != nil {
		return fmt.Errorf("error marking album '%s' tracks as fetched: %w", albumSpotifyID, err)
	}
	return nil
}

func (db *DB) MarkArtistAlbumsFetched(artistSpotifyID string) error {
	if err := db.
		Table("artists").
		Where("spotify_id = ?", artistSpotifyID).
		Update("has_fetched_albums", true).
		Error; err != nil {
		return fmt.Errorf("error marking artist '%s' albums as fetched: %w", artistSpotifyID, err)
	}
	return nil
}

func (db *DB) MarkGenreFetched(genreName string) error {
	if err := db.
		Table("genres").
		Where("name = ?", genreName).
		Update("has_fetched_artists", true).
		Error; err != nil {
		return fmt.Errorf("error marking genre '%s' as fetched: %w", genreName, err)
	}
	return nil
}

func (db *DB) MarkArtistFetched(artistSpotifyID string) error {
	if err := db.
		Table("artists").
		Where("spotify_id = ?", artistSpotifyID).
		Update("has_fetched_tracks", true).
		Error; err != nil {
		return fmt.Errorf("error marking artist '%s' as fetched: %w", artistSpotifyID, err)
	}
	return nil
}

func (db *DB) AddTrackAnalysis(track *data.Track) error {
	if err := db.Table("tracks").
		Where("spotify_id = ?", track.SpotifyID).
		Updates(map[string]interface{}{
			// NOTE:
			"has_analysis": true,

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

func (db *DB) InsertAlbum(album *data.Album) error {
	if err := db.
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(album).
		Error; err != nil {
		return fmt.Errorf("error inserting album '%s': %w", album.Name, err)
	}
	for _, artist := range album.Artists {
		if err := db.
			Clauses(clause.OnConflict{DoNothing: true}).
			Create(artist).
			Error; err != nil {
			return fmt.Errorf("error inserting artist '%s': %w", artist.Name, err)
		}
		if err := db.
			Clauses(clause.OnConflict{DoNothing: true}).
			Create(&data.AlbumArtist{
				AlbumSpotifyID:  album.SpotifyID,
				ArtistSpotifyID: artist.SpotifyID,
			}).
			Error; err != nil {
			return fmt.Errorf("error inserting album artist {'%s' '%s'}: %w", album.Name, artist.Name, err)
		}
	}
	return nil
}

func (db *DB) InsertTrack(track *data.Track) error {
	if err := db.
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(track).
		Error; err != nil {
		return fmt.Errorf("error inserting track '%s': %w", track.Name, err)
	}

	if err := db.
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(data.Album{
			SpotifyID: track.AlbumSpotifyID,
			Name:      track.AlbumName,
		}).
		Error; err != nil {
		return fmt.Errorf("error inserting album '%s': %w", track.AlbumName, err)
	}

	if err := db.
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&data.AlbumTrack{
			TrackSpotifyID: track.SpotifyID,
			AlbumSpotifyID: track.AlbumSpotifyID,
		}).
		Error; err != nil {
		return fmt.Errorf("error inserting track album {'%s', '%s'}: %w", track.Name, track.AlbumName, err)
	}

	for _, artist := range track.Artists {
		if err := db.
			Clauses(clause.OnConflict{DoNothing: true}).
			Create(artist).
			Error; err != nil {
			return fmt.Errorf("error inserting artist '%s': %w", artist.Name, err)
		}
		if err := db.
			Clauses(clause.OnConflict{DoNothing: true}).
			Create(&data.TrackArtist{
				TrackSpotifyID:  track.SpotifyID,
				ArtistSpotifyID: artist.SpotifyID,
			}).
			Error; err != nil {
			return fmt.Errorf("error inserting track artist {'%s', '%s'}: %w", track.Name, artist.Name, err)
		}
	}

	return nil
}

// InsertArtist, given an Artist, inserts the appropriate data into the artists,
// artist_genres, and artists_rtree tables, doing nothing if they already exist.
func (db *DB) InsertArtist(artist *data.Artist) error {
	if err := db.
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(artist).
		Error; err != nil {
		return fmt.Errorf("error inserting artist '%s' into artists: %w", artist.Name, err)
	}

	var rowid int64
	if err := db.
		Table("artists").
		Select("rowid").
		Where("spotify_id = ?", artist.SpotifyID).
		Scan(&rowid).
		Error; err != nil {
		return fmt.Errorf("error fetching rowid for inserted artist '%s': %w", artist.Name, err)
	}

	var stats struct {
		MinEnergy, MaxEnergy                     int64
		MinDynamicVariation, MaxDynamicVariation int64
		MinInstrumentalness, MaxInstrumentalness int64
		MinOrganicness, MaxOrganicness           int64
		MinBounciness, MaxBounciness             int64
	}

	if err := db.
		Table("genres").
		Where("name in ?", artist.Genres).
		Select(
			"min(energy) as min_energy",
			"max(energy) as max_energy",

			"min(dynamic_variation) as min_dynamic_variation",
			"max(dynamic_variation) as max_dynamic_variation",

			"min(instrumentalness) as min_instrumentalness",
			"max(instrumentalness) as max_instrumentalness",

			"min(organicness) as min_organicness",
			"max(organicness) as max_organicness",

			"min(bounciness) as min_bounciness",
			"max(bounciness) as max_bounciness",
		).
		Scan(&stats).
		Error; err != nil {
		return fmt.Errorf("error fetching genre stats while inserting artist '%s': %w", artist.Name, err)
	}

	if err := db.Exec(`insert or ignore into artists_rtree values (?, ?,?, ?,?, ?,?, ?,?, ?,?)`,
		rowid,
		stats.MinEnergy, stats.MaxEnergy,
		stats.MinDynamicVariation, stats.MaxDynamicVariation,
		stats.MinInstrumentalness, stats.MaxInstrumentalness,
		stats.MinOrganicness, stats.MaxOrganicness,
		stats.MinBounciness, stats.MaxBounciness,
	).Error; err != nil {
		return fmt.Errorf("error inserting stats into artists_rtree for artist '%s': %w", artist.Name, err)
	}

	for _, genre := range artist.Genres {
		if err := db.
			Clauses(clause.OnConflict{DoNothing: true}).
			Create(&data.ArtistGenre{
				ArtistSpotifyID: artist.SpotifyID,
				GenreName:       genre,
			}).
			Error; err != nil {
			return fmt.Errorf("error inserting artist_genre for artist '%s' and genre '%s': %w", artist.Name, genre, err)
		}
	}

	return nil
}
