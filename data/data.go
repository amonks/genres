package data

import "time"

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

	FetchedArtistsAt *time.Time
}

// Artists holds the artists we've found using Spotify's search API. We try to
// fetch the thousand first artists returned by a search for each genre.
//
// Artists have many genres via the association table artist_genres.
//
// Artists also have an rtree representing their bounds in genre-space. The 'id'
// column in artists_rtree references the automatically-generated 'rowid' column
// in artists.
type Artist struct {
	SpotifyID  string
	ImageURL   string
	Name       string
	Followers  int64
	Popularity int64
	Genres     []string `gorm:"-"`

	FetchedTracksAt *time.Time
	FetchedAlbumsAt *time.Time
}

type Album struct {
	SpotifyID string
	Name      string
	Type      string
	ImageURL  string

	TotalTracks     int64
	FetchedTracksAt *time.Time

	ReleaseDate          string
	ReleaseDatePrecision string

	Artists []Artist `gorm:"-"`
	Tracks  []Track  `gorm:"-"`
}

type AlbumArtist struct {
	ArtistSpotifyID string
	AlbumSpotifyID  string
}

type AlbumTrack struct {
	TrackSpotifyID string
	AlbumSpotifyID string
}

// artists_rtree stores the bounds of each artist within 5-dimensional genre
// space, and can be used for geospatial-style range queries.
//
// For each artist, spotify gives us a list of genres. We can imagine finding
// all of those genres on the everynoise website and drawing a bounding box
// around them. Then, we can point to a spot on the visualization and query for
// the artists whose boxes contain that point.
//
// But actually: the visualization has more than two dimensions. In addition to
// x and y position, each genre has a color, and the three channels of that
// color (RGB) represent additional data.
type ArtistRtree struct {
	ID int64

	MinEnergy, MaxEnergy                     int64
	MinDynamicVariation, MaxDynamicVariation int64
	MinInstrumentalness, MaxInstrumentalness int64
	MinOrganicness, MaxOrganicness           int64
	MinBounciness, MaxBounciness             int64
}

// ArtistGenres represent a many-to-many relationship between artists and
// genres.
type ArtistGenre struct {
	ArtistSpotifyID string
	GenreName       string
}

type Track struct {
	SpotifyID  string
	Name       string
	Popularity int64

	AlbumSpotifyID string
	AlbumName      string
	DiscNumber     int64
	TrackNumber    int64

	Artists []Artist `gorm:"-"`

	FetchedAnalysisAt *time.Time
	FailedAnalysisAt  *time.Time
	IndexedSearchAt   *time.Time

	Key           int64
	Mode          int64
	Tempo         float64
	TimeSignature int64

	Acousticness     float64
	Danceability     float64
	Energy           float64
	Instrumentalness float64
	Liveness         float64
	Loudness         float64
	Speechiness      float64
	Valence          float64
}

func (t *Track) Vector() Vector {
	if t.FetchedAnalysisAt == nil {
		return Vector{}
	}
	return Vector{
		"acousticness":     t.Acousticness,
		"danceability":     t.Danceability,
		"energy":           t.Energy,
		"instrumentalness": t.Instrumentalness,
		"liveness":         t.Liveness,
		"speechiness":      t.Speechiness,
		"valence":          t.Valence,
	}
}

type TrackArtist struct {
	TrackSpotifyID  string
	ArtistSpotifyID string
}

type TracksSearch struct {
	TrackSpotifyID string
	Content        string
}
