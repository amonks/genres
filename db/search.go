package db

import (
	"context"
	"fmt"
	"strings"

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
		Where("has_search = false").
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
		Where("has_search = false").
		Limit(limit).
		Pluck("spotify_id", &ids).
		Error; err != nil {
		return nil, fmt.Errorf("error getting %d tracks where has_search=false: %w", limit, err)
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
	inserts := make([]data.TracksSearch, len(tracks))
	ids := make([]string, len(tracks))
	for i, track := range tracks {
		var content []string
		content = append(content, track.Name)
		content = append(content, track.AlbumName)
		for _, artist := range track.Artists {
			content = append(content, artist.Name)
		}
		inserts[i] = data.TracksSearch{
			TrackSpotifyID: track.SpotifyID,
			Content:        strings.Join(content, "\n"),
		}
		ids[i] = track.SpotifyID
	}

	defer db.hold()()

	return db.rw.Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Table("tracks_search").
			Create(inserts).
			Error; err != nil {
			return fmt.Errorf("error inserting %d tracks into search index: %w", len(tracks), err)
		}

		if err := ctx.Err(); err != nil {
			return fmt.Errorf("canceled: %w", err)
		}

		if err := tx.
			Table("tracks").
			Where("spotify_id in ?", ids).
			Update("has_search", true).
			Error; err != nil {
			return fmt.Errorf("error marking %d tracks as indexed: %w", len(ids), err)
		}

		return nil
	})
}
