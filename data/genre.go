package data

import (
	"database/sql"
)

// Genres holds the list of genres extracted from everynoise.com.
//
// Genres have many artists via the association table artist_genres.
type Genre struct {
	// like "pop"
	Name string

	// like "3nzVSyaYk0KNrahyNQS0Ur"
	Key string

	// Like `Budapest Chorus "Let the Light Shine on Me"`
	//
	// We can't safely parse this into artist/track, because quote marks
	// -within- artist and track names are not guaranteed to be matched
	// properly.
	Example string

	// A value in the range [0, 1], derived from the ENAO visualization.
	Energy, DynamicVariation, Instrumentalness, Organicness, Bounciness, Popularity float64

	FetchedArtistsAt sql.NullTime
}
