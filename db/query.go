//go:build sqlite_math_functions && fts5

package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/amonks/genres/data"
)

// approach:
// let's say our query is {"energy": 0.4, "liveness": 0.3}
//  1. we set a threshold value, `epsilon`
//  2. we query for tracks whose energy and liveness are _both_ within `epsilon`
//     of the input values, sorted by distance
//  3. if we didn't get enough, we repeat with a larger `epsilon`.
func (db *DB) NearestTracks(ctx context.Context, count int, input data.Vector) ([]data.Track, error) {
	var ids []string
	for epsilon := 0.01; ; epsilon *= 2 {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("canceled: %w", err)
		}

		var terms []string
		for k, v := range input {
			terms = append(terms, fmt.Sprintf("pow(tracks.%s - %f, 2.0)", k, v))
		}
		q := db.ro.
			Table("tracks").
			Where("has_analysis = true").
			Order(fmt.Sprintf("sqrt(%s) asc", strings.Join(terms, " + "))).
			Limit(count)
		for k, v := range input {
			q = q.Where(fmt.Sprintf("%s between %f and %f", k, v-epsilon, v+epsilon))
		}
		if err := q.Pluck("spotify_id", &ids).Error; err != nil {
			return nil, fmt.Errorf("error finding tracks: %w", err)
		}
		if len(ids) == count {
			break
		}
	}

	tracks := make([]data.Track, len(ids))
	for i, id := range ids {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("canceled: %w", err)
		}

		track, err := db.GetTrack(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("error getting track %d ('%s'): %w", i+1, id, err)
		}
		tracks[i] = *track
	}

	return tracks, nil
}

func (db *DB) GetTrack(ctx context.Context, id string) (*data.Track, error) {
	var track data.Track
	if err := db.ro.
		Table("tracks").
		Where("spotify_id = ?", id).
		First(&track).
		Error; err != nil {
		return nil, fmt.Errorf("error getting track '%s': %w", id, err)
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("canceled: %w", err)
	}

	var trackArtistIDs []string
	if err := db.ro.
		Table("track_artists").
		Where("track_spotify_id = ?", id).
		Pluck("artist_spotify_id", &trackArtistIDs).
		Error; err != nil {
		return nil, fmt.Errorf("error getting artist IDs for track '%s': %w", id, err)
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("canceled: %w", err)
	}

	track.Artists = make([]data.Artist, len(trackArtistIDs))
	artistCache := map[string]*data.Artist{}
	for i, artistID := range trackArtistIDs {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("canceled: %w", err)
		}

		if artist, has := artistCache[artistID]; has {
			track.Artists[i] = *artist
			continue
		}
		artist, err := db.GetArtist(artistID)
		if err != nil {
			return nil, fmt.Errorf("error getting artist '%s': %w", artistID, err)
		}
		artistCache[artistID] = artist
		track.Artists[i] = *artist
	}
	return &track, nil
}

func (db *DB) GetArtist(id string) (*data.Artist, error) {
	var artist data.Artist
	if err := db.ro.
		Table("artists").
		Where("spotify_id = ?", id).
		First(&artist).
		Error; err != nil {
		return nil, fmt.Errorf("error getting artist '%s': %w", id, err)
	}
	var genres []string
	if err := db.ro.
		Table("artist_genres").
		Where("artist_spotify_id = ?", id).
		Pluck("genre_name", &genres).
		Error; err != nil {
		return nil, fmt.Errorf("error getting genres for artist '%s': %w", id, err)
	}
	artist.Genres = genres
	return &artist, nil
}
