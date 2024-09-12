package data

import "database/sql"

// Albums are Fetched from Spotify.
type Album struct {
	SpotifyID string
	Name      string
	Type      string
	ImageURL  string

	TotalTracks int64

	Popularity int64

	ReleaseDate          string
	ReleaseDatePrecision string

	Artists []Artist `gorm:"-"`
	Tracks  []Track  `gorm:"-"`
	Genres  []string `gorm:"-"`

	FetchedTracksAt      sql.NullTime
	FailedTracksAt       sql.NullTime
	IndexedTracksRtreeAt sql.NullTime
}
