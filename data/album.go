package data

import "database/sql"

// Albums are Fetched from Spotify.
type Album struct {
	SpotifyID string
	Name      string
	Type      string
	ImageURL  string

	TotalTracks int64

	ReleaseDate          string
	ReleaseDatePrecision string

	Artists []Artist `gorm:"-"`
	Tracks  []Track  `gorm:"-"`

	FetchedTracksAt      sql.NullTime
	IndexedTracksRtreeAt sql.NullTime
}
