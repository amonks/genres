package db

import "fmt"

func (db *DB) CountGenresKnown() (int, error) {
	var count int64
	if err := db.ro.
		Table("genres").
		Count(&count).
		Error; err != nil {
		return 0, fmt.Errorf("error counting genres: %w", err)
	}
	return int(count), nil
}

func (db *DB) CountGenresWithFetchedArtists() (int, error) {
	var count int64
	if err := db.ro.
		Table("genres").
		Where("fetched_artists_at is not null").
		Count(&count).
		Error; err != nil {
		return 0, fmt.Errorf("error counting genres with fetched artists: %w", err)
	}
	return int(count), nil
}

func (db *DB) CountGenresToFetchArtists() (int, error) {
	var count int64
	if err := db.ro.
		Table("genres").
		Where("fetched_artists_at is null").
		Count(&count).
		Error; err != nil {
		return 0, fmt.Errorf("error counting genres that need to fetch artists: %w", err)
	}
	return int(count), nil
}

func (db *DB) GetGenresToFetchArtists(limit int) ([]string, error) {
	genreNames := []string{}
	if err := db.ro.
		Table("genres").
		Limit(limit).
		Where("fetched_artists_at is null").
		Pluck("name", &genreNames).
		Error; err != nil {
		return nil, err
	}
	return genreNames, nil
}

func (db *DB) CountArtistsWithFetchedTracks() (int, error) {
	var count int64
	if err := db.ro.
		Table("artists").
		Where("fetched_tracks_at is not null").
		Count(&count).
		Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (db *DB) CountArtistsToFetchTracks() (int, error) {
	var count int64
	if err := db.ro.
		Table("artists").
		Where("fetched_tracks_at is null").
		Count(&count).
		Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (db *DB) GetArtistsToFetchTracks(limit int) ([]string, error) {
	artists := []string{}
	if err := db.ro.
		Table("artists").
		Limit(limit).
		Where("fetched_tracks_at is null").
		Pluck("spotify_id", &artists).
		Error; err != nil {
		return nil, err
	}
	return artists, nil
}

func (db *DB) CountArtistsToFetchAlbums() (int, error) {
	var count int64
	if err := db.ro.
		Table("artists").
		Where("fetched_albums_at is null").
		Count(&count).
		Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (db *DB) CountArtistsWithFetchedAlbums() (int, error) {
	var count int64
	if err := db.ro.
		Table("artists").
		Where("fetched_albums_at is not null").
		Count(&count).
		Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (db *DB) GetArtistsToFetchAlbums(limit int) ([]string, error) {
	artists := []string{}
	if err := db.ro.
		Table("artists").
		Limit(limit).
		Where("fetched_albums_at is null").
		Pluck("spotify_id", &artists).
		Error; err != nil {
		return nil, err
	}
	return artists, nil
}

func (db *DB) CountAlbumsToFetchTracks() (int, error) {
	var count int64
	if err := db.ro.
		Table("albums").
		Where("fetched_tracks_at is null").
		Count(&count).
		Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (db *DB) CountAlbumsWithFetchedTracks() (int, error) {
	var count int64
	if err := db.ro.
		Table("albums").
		Where("fetched_tracks_at is not null").
		Count(&count).
		Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (db *DB) GetAlbumsToFetchTracks(limit int) ([]string, error) {
	albums := []string{}
	if err := db.ro.
		Table("albums").
		Limit(limit).
		Where("fetched_tracks_at is null").
		Pluck("spotify_id", &albums).
		Error; err != nil {
		return nil, err
	}
	return albums, nil
}

func (db *DB) CountTracksWithFetchedAnalysis() (int, error) {
	var count int64
	if err := db.ro.
		Table("tracks").
		Where("fetched_analysis_at is not null or failed_analysis_at is not null").
		Count(&count).
		Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (db *DB) CountTracksToFetchAnalysis() (int, error) {
	var count int64
	if err := db.ro.
		Table("tracks").
		Where("fetched_analysis_at is null and failed_analysis_at is null").
		Count(&count).
		Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (db *DB) GetTracksToFetchAnalysis(limit int) ([]string, error) {
	tracks := []string{}
	if err := db.ro.
		Table("tracks").
		Limit(limit).
		Where("fetched_analysis_at is null and failed_analysis_at is null").
		Pluck("spotify_id", &tracks).
		Error; err != nil {
		return nil, err
	}
	return tracks, nil
}

func (db *DB) CountArtistsKnown() (int, error) {
	var count int64
	if err := db.ro.
		Table("artists").
		Count(&count).
		Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (db *DB) CountArtistAlbumsDone() (int, error) {
	var count int64
	if err := db.ro.
		Table("artists").
		Where("fetched_albums_at is not null").
		Count(&count).
		Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (db *DB) CountArtistTracksDone() (int, error) {
	var count int64
	if err := db.ro.
		Table("artists").
		Where("fetched_tracks_at is not null").
		Count(&count).
		Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (db *DB) CountArtistsDone() (int, error) {
	var count int64
	if err := db.ro.
		Table("artists").
		Where("fetched_tracks_at is not null").
		Where("fetched_albums_at is not null").
		Count(&count).
		Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (db *DB) CountAlbumsKnown() (int, error) {
	var count int64
	if err := db.ro.
		Table("albums").
		Count(&count).
		Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (db *DB) CountAlbumsDone() (int, error) {
	var count int64
	if err := db.ro.
		Table("albums").
		Where("fetched_tracks_at is not null").
		Count(&count).
		Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (db *DB) CountTracksKnown() (int, error) {
	var count int64
	if err := db.ro.
		Table("tracks").
		Count(&count).
		Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (db *DB) CountTracksDone() (int, error) {
	var count int64
	if err := db.ro.
		Table("tracks").
		Where("fetched_analysis_at is not null or failed_analysis_at is not null").
		Count(&count).
		Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (db *DB) CountTracksIndexed() (int, error) {
	var count int64
	if err := db.ro.
		Table("tracks").
		Where("indexed_search_at is not null").
		Count(&count).
		Error; err != nil {
		return 0, err
	}
	return int(count), nil
}
