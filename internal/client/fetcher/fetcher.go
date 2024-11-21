package fetcher

// 이후, 테스트를 위한 링크
// 1.
// https://nasa-i.akamaihd.net/hls/live/253565/NASA-NTV1-Public/master.m3u8
// Header 변경: Referer: https://www.nasa.gov/multimedia/nasatv/
// 2.
// http://commondatastorage.googleapis.com/gtv-videos-bucket/sample/BigBuckBunny.mp4

import (
	"fmt"
	"net/http"
)

type HTTPVideoFetcher struct {
	client *http.Client
}

func NewHTTPVideoFetcher() *HTTPVideoFetcher {
	return &HTTPVideoFetcher{
		client: &http.Client{},
	}
}

func (f *HTTPVideoFetcher) Fetch(url string) (*VideoResponse, error) {
	resp, err := f.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch video: %w", err)
	}

	return &VideoResponse{
		Body:        resp.Body,
		Headers:     resp.Header,
		ContentType: resp.Header.Get("Content-Type"),
	}, nil

}
