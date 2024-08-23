package data

import "database/sql"

type Track struct {
	SpotifyID  string
	Name       string
	Popularity int64

	AlbumSpotifyID string
	AlbumName      string
	DiscNumber     int64
	TrackNumber    int64

	Artists []Artist `gorm:"-"`

	FetchedAnalysisAt sql.NullTime
	FailedAnalysisAt  sql.NullTime
	IndexedSearchAt   sql.NullTime

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
	if !t.FetchedAnalysisAt.Valid {
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

type TracksSearch struct {
	SpotifyID   string
	Name        string
	AlbumName   string
	ArtistNames string
	Popularity  float64
}
