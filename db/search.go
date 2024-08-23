package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/amonks/genres/data"
	"gorm.io/gorm"
)

func (db *DB) Search(ctx context.Context, query string, limit int) ([]data.Track, error) {
	var ids []string
	if err := db.ro.
		Table("tracks_search").
		Where("content match ?", query).
		Order("rank").
		Limit(limit).
		Pluck("track_spotify_id", &ids).
		Error; err != nil {
		return nil, err
	}
	tracks := make([]data.Track, len(ids))
	for i, id := range ids {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("canceled: %w", err)
		}

		track, err := db.GetTrack(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("error getting track '%s': %w", id, err)
		}
		tracks[i] = *track
	}
	return tracks, nil
}

func (db *DB) CountTracksToIndex(ctx context.Context) (int, error) {
	var count int64
	if err := db.ro.
		Table("tracks").
		Where("indexed_search_at is null").
		Count(&count).
		Error; err != nil {
		return 0, fmt.Errorf("error counting tracks to index: %w", err)
	}
	return int(count), nil
}

func (db *DB) GetTracksToIndex(ctx context.Context, limit int) ([]data.Track, error) {
	var ids []string
	if err := db.ro.
		Table("tracks").
		Where("indexed_search_at is null").
		Limit(limit).
		Pluck("spotify_id", &ids).
		Error; err != nil {
		return nil, fmt.Errorf("error getting %d tracks where indexed_search_at is null: %w", limit, err)
	}
	if len(ids) == 0 {
		return nil, nil
	}
	tracks := make([]data.Track, len(ids))
	for i, id := range ids {
		track, err := db.GetTrack(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("error getting track '%s': %w", id, err)
		}
		tracks[i] = *track
	}
	return tracks, nil
}

func (db *DB) IndexTracks(ctx context.Context, tracks []data.Track) error {
	ids := make([]string, len(tracks))
	for i, track := range tracks {
		ids[i] = track.SpotifyID
	}

	defer db.hold()()

	return db.rw.Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Exec("insert into tracks_search select * from tracks_with_artist_names where spotify_id in ?", ids).
			Error; err != nil {
			return fmt.Errorf("error inserting %d tracks into search index: %w", len(tracks), err)
		}

		if err := ctx.Err(); err != nil {
			return fmt.Errorf("canceled: %w", err)
		}

		if err := tx.
			Table("tracks").
			Where("spotify_id in ?", ids).
			Update("indexed_search_at", sql.NullTime{Time: time.Now(), Valid: true}).
			Error; err != nil {
			return fmt.Errorf("error marking %d tracks as indexed: %w", len(ids), err)
		}

		return nil
	})
}
