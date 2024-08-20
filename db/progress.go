package db

import "fmt"

func (db *DB) CountGenresToFetchArtists() (int, error) {
	var count int64
	if err := db.
		Table("genres").
		Where("has_fetched_artists = false").
		Count(&count).
		Error; err != nil {
		return 0, fmt.Errorf("error counting genres that need to fetch artists: %w", err)
	}
	return int(count), nil
}

func (db *DB) GetGenresToFetchArtists(limit int) ([]string, error) {
	genreNames := []string{}
	if err := db.
		Table("genres").
		Limit(limit).
		Where("has_fetched_artists = false").
		Pluck("name", &genreNames).
		Error; err != nil {
		return nil, err
	}
	return genreNames, nil
}

func (db *DB) CountArtistsToFetchTracks() (int, error) {
	var count int64
	if err := db.
		Table("artists").
		Where("has_fetched_tracks = false").
		Count(&count).
		Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (db *DB) GetArtistsToFetchTracks(limit int) ([]string, error) {
	artists := []string{}
	if err := db.
		Table("artists").
		Limit(limit).
		Where("has_fetched_tracks = false").
		Pluck("spotify_id", &artists).
		Error; err != nil {
		return nil, err
	}
	return artists, nil
}

func (db *DB) CountArtistsToFetchAlbums() (int, error) {
	var count int64
	if err := db.
		Table("artists").
		Where("has_fetched_albums = false").
		Count(&count).
		Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (db *DB) GetArtistsToFetchAlbums(limit int) ([]string, error) {
	artists := []string{}
	if err := db.
		Table("artists").
		Limit(limit).
		Where("has_fetched_albums = false").
		Pluck("spotify_id", &artists).
		Error; err != nil {
		return nil, err
	}
	return artists, nil
}

func (db *DB) CountAlbumsToFetchTracks() (int, error) {
	var count int64
	if err := db.
		Table("albums").
		Where("has_fetched_tracks = false").
		Count(&count).
		Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (db *DB) GetAlbumsToFetchTracks(limit int) ([]string, error) {
	albums := []string{}
	if err := db.
		Table("albums").
		Limit(limit).
		Where("has_fetched_tracks = false").
		Pluck("spotify_id", &albums).
		Error; err != nil {
		return nil, err
	}
	return albums, nil
}

func (db *DB) CountTracksToFetchAnalysis() (int, error) {
	var count int64
	if err := db.
		Table("tracks").
		Where("has_analysis = false").
		Count(&count).
		Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (db *DB) GetTracksToFetchAnalysis(limit int) ([]string, error) {
	tracks := []string{}
	if err := db.
		Table("tracks").
		Limit(limit).
		Where("has_analysis = false").
		Where("failed_analysis = false").
		Pluck("spotify_id", &tracks).
		Error; err != nil {
		return nil, err
	}
	return tracks, nil
}

func (db *DB) CountArtistsKnown() (int, error) {
	var count int64
	if err := db.
		Table("artists").
		Count(&count).
		Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (db *DB) CountArtistsDone() (int, error) {
	var count int64
	if err := db.
		Table("artists").
		Where("has_fetched_tracks = true").
		Where("has_fetched_albums = true").
		Count(&count).
		Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (db *DB) CountAlbumsKnown() (int, error) {
	var count int64
	if err := db.
		Table("albums").
		Count(&count).
		Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (db *DB) CountAlbumsDone() (int, error) {
	var count int64
	if err := db.
		Table("albums").
		Where("has_fetched_tracks = true").
		Count(&count).
		Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (db *DB) CountTracksKnown() (int, error) {
	var count int64
	if err := db.
		Table("tracks").
		Count(&count).
		Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (db *DB) CountTracksDone() (int, error) {
	var count int64
	if err := db.
		Table("tracks").
		Where("has_analysis = true").
		Count(&count).
		Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (db *DB) CountTracksIndexed() (int, error) {
	var count int64
	if err := db.
		Table("tracks").
		Where("has_search = true").
		Count(&count).
		Error; err != nil {
		return 0, err
	}
	return int(count), nil
}
