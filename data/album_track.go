package data

type AlbumTrack struct {
	TrackSpotifyID string
	AlbumSpotifyID string
}

// AlbumTracksRtree stores the bounds of each album's tracks within
// 5-dimensional genre space, and can be used for geospatial-style range
// queries.
type AlbumTracksRtree struct {
	ID int64

	MinEnergy, MaxEnergy                     int64
	MinDynamicVariation, MaxDynamicVariation int64
	MinInstrumentalness, MaxInstrumentalness int64
	MinOrganicness, MaxOrganicness           int64
	MinBounciness, MaxBounciness             int64
}
