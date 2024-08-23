package data

import "database/sql"

// Artists holds the artists we've found using Spotify's search API.
type Artist struct {
	SpotifyID  string
	ImageURL   string
	Name       string
	Followers  int64
	Popularity int64
	Genres     []string `gorm:"-"`

	FetchedTracksAt      sql.NullTime
	FetchedAlbumsAt      sql.NullTime
	IndexedGenresRtreeAt sql.NullTime
	IndexedTracksRtreeAt sql.NullTime
}
