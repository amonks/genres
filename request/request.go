package request

import (
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/PuerkitoBio/goquery"
)

// FetchHTML does an HTTP GET on the given URL, then parses the response as
// HTML.
func FetchHTML(url string) (*goquery.Document, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching '%s': %w", url, err)
	}
	if err := Error(resp); err != nil {
		return nil, fmt.Errorf("unexpected status from '%s': %w", url, err)
	}

	if contentType := resp.Header.Get("Content-type"); contentType != "text/html" {
		return nil, fmt.Errorf("expected an html response at '%s', but got '%s'", url, contentType)
	}

	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error parsing html from '%s': %w", url, err)
	}

	return doc, nil
}

// Error checks the given http response for an error code, and, if one is
// present, reads the body and returns a friendly error.
func Error(resp *http.Response) error {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bs, err := httputil.DumpResponse(resp, true)
		if err != nil {
			return fmt.Errorf("http status code %d; error decoding body: %w", resp.StatusCode, err)
		} else {
			return fmt.Errorf("http status code %d:\n%s", resp.StatusCode, string(bs))
		}
	}
	return nil
}
