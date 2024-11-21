package fetcher

import (
	"io"
)

type VideoFetcher interface {
	Fetch(url string) (*VideoResponse, error)
}

type VideoResponse struct {
	Body        io.ReadCloser
	Headers     map[string][]string
	ContentType string
}
