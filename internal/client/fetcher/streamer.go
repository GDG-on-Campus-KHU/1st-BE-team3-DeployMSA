package fetcher

import (
	"fmt"
	"io"
	"net/http"
)

type VideoStreamer struct {
	fetcher    VideoFetcher
	bufferSize int
}

func NewVideoStreamer(fetcher VideoFetcher) *VideoStreamer {
	return &VideoStreamer{
		fetcher:    fetcher,
		bufferSize: 32 * 1024, // 32KB buffer,
	}
}

func (s *VideoStreamer) StreamVideo(w http.ResponseWriter, videoURL string) error {
	response, err := s.fetcher.Fetch(videoURL)
	if err != nil {
		return fmt.Errorf("failed to fetch video: %w", err)
	}
	defer response.Body.Close()

	// Copy headers
	for k, v := range response.Headers {
		w.Header()[k] = v
	}
	w.Header().Set("Content-Type", response.ContentType)

	// Stream in chunks
	buffer := make([]byte, s.bufferSize)
	for {
		n, err := response.Body.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading chunk: %w", err)
		}
		if _, err := w.Write(buffer[:n]); err != nil {
			return fmt.Errorf("error writing chunk: %w", err)
		}

		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
	return nil
}
