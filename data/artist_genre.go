package data

// An ArtistGenre represents a many-to-many relationship between artists and
// genres.
type ArtistGenre struct {
	ArtistSpotifyID string
	GenreName       string
}

// ArtistGenresRtree stores the bounds of each artist's genres within
// 5-dimensional genre space, and can be used for geospatial-style range
// queries.
type ArtistGenresRtree struct {
	ID int64

	MinEnergy, MaxEnergy                     int64
	MinDynamicVariation, MaxDynamicVariation int64
	MinInstrumentalness, MaxInstrumentalness int64
	MinOrganicness, MaxOrganicness           int64
	MinBounciness, MaxBounciness             int64
}
