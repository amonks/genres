package enao

import (
	"fmt"
	"strconv"

	"github.com/amonks/genres/data"
)

// AllGenres extracts all the information we can out of the visualization at
// everynoise.com.
func AllGenres() ([]*data.Genre, error) {
	visualization, err := FetchVisualization()
	if err != nil {
		return nil, fmt.Errorf("error fetching enao visualization: %w", err)
	}

	return visualization.ToGenres(), nil
}

// A Visualization represents all the data we can extract from the page at
// everynoise.com. This includes the set of genres and how they are visualized
// (eg their placement on the screen), and it also includes metadata about the
// visualization itself, like the _range of_ positions where genres are placed.
type Visualization struct {
	Genres []Genre

	MinTop, MaxTop           int64
	MinLeft, MaxLeft         int64
	MinRed, MaxRed           int64
	MinGreen, MaxGreen       int64
	MinBlue, MaxBlue         int64
	MinFontSize, MaxFontSize int64
}

// NewVisualization, given a set of genres extracted from everynoise.com,
// creates a Visualization, computing metadata like the range of positions where
// genres are placed.
func NewVisualization(genres []Genre) *Visualization {
	vis := &Visualization{Genres: genres}

	vis.MinTop, vis.MinLeft, vis.MinRed, vis.MinGreen, vis.MinBlue, vis.MinFontSize = -1, -1, -1, -1, -1, -1
	for _, genre := range vis.Genres {
		if genre.Top < vis.MinTop || vis.MinTop < 0 {
			vis.MinTop = genre.Top
		}
		if genre.Top > vis.MaxTop {
			vis.MaxTop = genre.Top
		}
		if genre.Left < vis.MinLeft || vis.MinLeft < 0 {
			vis.MinLeft = genre.Left
		}
		if genre.Left > vis.MaxLeft {
			vis.MaxLeft = genre.Left
		}
		if genre.Red() < vis.MinRed || vis.MinRed < 0 {
			vis.MinRed = genre.Red()
		}
		if genre.Red() > vis.MaxRed {
			vis.MaxRed = genre.Red()
		}
		if genre.Green() < vis.MinGreen || vis.MinGreen < 0 {
			vis.MinGreen = genre.Green()
		}
		if genre.Green() > vis.MaxGreen {
			vis.MaxGreen = genre.Green()
		}
		if genre.Blue() < vis.MinBlue || vis.MinBlue < 0 {
			vis.MinBlue = genre.Blue()
		}
		if genre.Blue() > vis.MaxBlue {
			vis.MaxBlue = genre.Blue()
		}
		if genre.FontSize < vis.MinFontSize || vis.MinFontSize < 0 {
			vis.MinFontSize = genre.FontSize
		}
		if genre.FontSize > vis.MaxFontSize {
			vis.MaxFontSize = genre.FontSize
		}
	}

	return vis
}

// ToGenres converts a visualization into a list of Genres, normalizing the
// visualization data (like a genre's color or position) back into echonest data
// (like energy and dynamic variation)
func (vis *Visualization) ToGenres() []*data.Genre {
	out := make([]*data.Genre, len(vis.Genres))
	for i, genre := range vis.Genres {
		out[i] = &data.Genre{
			Name:       genre.Name,
			Key:        genre.Key,
			Example:    genre.Example,

			Energy:           normalize(vis.MinRed, vis.MaxRed, genre.Red()),
			DynamicVariation: normalize(vis.MinGreen, vis.MaxGreen, genre.Green()),
			Instrumentalness: normalize(vis.MinBlue, vis.MaxBlue, genre.Blue()),
			Organicness:      normalize(vis.MinTop, vis.MaxTop, genre.Top),
			Bounciness:       normalize(vis.MinLeft, vis.MaxLeft, genre.Left),
			Popularity:       normalize(vis.MinFontSize, vis.MaxFontSize, genre.FontSize),
		}
	}

	return out
}

// A Genre represents a genre parsed from the visualization on the ENAO website.
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

	// From the rendering on the ENAO website. Like 389fb1.
	//
	// The red channel encodes energy, the green channel encodes dynamic
	// variation, and the blue channel encodes instrumentalness, according
	// to this blog post:
	//
	// https://www.furia.com/page.cgi?type=log&id=419
	Color string

	// From the rendering on the ENAO website. Y-position ("top") encodes
	// organicness, X-position ("left") encodes bounciness, Font size
	// encodes popularity.
	Top, Left, FontSize int64
}

// Extracts the red (energy) channel to a number [0, 255]
func (g Genre) Red() int64 { return hexToInt(g.Color[0:2]) }

// Extracts the green (dynamic variation) channel to a number [0, 255]
func (g Genre) Green() int64 { return hexToInt(g.Color[2:4]) }

// Extracts the blue (instrumentalness) channel to a number [0, 255]
func (g Genre) Blue() int64 { return hexToInt(g.Color[4:6]) }

func hexToInt(hex string) int64 {
	ui, err := strconv.ParseUint(hex, 16, 16)
	if err != nil {
		panic(err)
	}
	return int64(ui)
}

func normalize(min, max, value int64) int64 {
	return int64(float64(value-min) / float64(max-min) * 4096)
}
