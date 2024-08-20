package enao

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/amonks/genres/request"
)

const allGenresURL = "https://everynoise.com"

// FetchVisualization requests the html page at everynoise.com, parses the html
// to extract a list of genres, then converts them into a form that's nice to
// work with.
func FetchVisualization() (*Visualization, error) {
	doc, err := request.FetchHTML(allGenresURL)
	if err != nil {
		return nil, err
	}

	var genres []Genre
	var findErr error
	doc.Find("div.canvas > div").Each(func(i int, sel *goquery.Selection) {
		if findErr != nil {
			return
		}
		genre, err := genreElement{sel}.Genre()
		if err != nil {
			findErr = err
			return
		}
		genres = append(genres, genre)
	})
	if findErr != nil {
		return nil, findErr
	}

	return NewVisualization(genres), nil
}

// A genreElement is the div for a single genre on everynoise.com. It has
// methods for looking into that div and extracting information.
type genreElement struct{ *goquery.Selection }

func (el genreElement) Genre() (Genre, error) {
	var genre Genre
	var err error
	genre.Name = el.Name()
	if genre.Key, err = el.Key(); err != nil {
		return genre, err
	}
	if genre.Color, genre.Top, genre.Left, genre.FontSize, err = el.Data(); err != nil {
		return genre, err
	}
	if genre.Example, err = el.Example(); err != nil {
		return genre, err
	}
	return genre, nil
}

func (el genreElement) Name() string {
	return strings.TrimSuffix(el.Text(), "» ")
}

var keyRE = regexp.MustCompile(`^playx\("(?P<Key>\w+)", ".+", this\);$`)

func (el genreElement) Key() (string, error) {
	onclick, found := el.Attr("onclick")
	if !found {
		return "", fmt.Errorf("genre '%s' has no onclick attribute", el.Name())
	}
	match := keyRE.FindStringSubmatch(onclick)
	return match[1], nil
}

var styleRE = regexp.MustCompile(`^color: #(\w{6}); top: (\d+)px; left: (\d+)px; font-size: (\d+)%$`)

func (el genreElement) Data() (string, int64, int64, int64, error) {
	style, found := el.Attr("style")
	if !found {
		return "", 0, 0, 0, fmt.Errorf("genre '%s' has no style attribute", el.Name())
	}
	match := styleRE.FindStringSubmatch(style)
	color, topStr, leftStr, fontSizeStr := match[1], match[2], match[3], match[4]

	top, err := strconv.ParseInt(topStr, 10, 64)
	if err != nil {
		return "", 0, 0, 0, fmt.Errorf("error parsing 'top' from genre '%s': %w", el.Name(), err)
	}
	left, err := strconv.ParseInt(leftStr, 10, 64)
	if err != nil {
		return "", 0, 0, 0, fmt.Errorf("error parsing 'left' from genre '%s': %w", el.Name(), err)
	}
	fontSize, err := strconv.ParseInt(fontSizeStr, 10, 64)
	if err != nil {
		return "", 0, 0, 0, fmt.Errorf("error parsing 'fontSize' from genre '%s': %w", el.Name(), err)
	}
	return color, top, left, fontSize, nil
}

func (el genreElement) Example() (string, error) {
	title, found := el.Attr("title")
	if !found {
		return "", fmt.Errorf("genre '%s' has no title attribute", el.Name())
	}
	return strings.TrimPrefix(title, "e.g. "), nil
}
