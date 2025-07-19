package http

import (
	"io"
	"net/http"
	"time"
)

// RequestParams holds parameters for HTTP requests
type RequestParams struct {
	URL     string
	Referer string
	Headers map[string]string
}

// Client is a custom HTTP client with default settings
var Client = &http.Client{
	Timeout: 30 * time.Second,
}

// Get performs a GET request with the given parameters
func Get(params RequestParams) (io.ReadCloser, error) {
	req, err := http.NewRequest("GET", params.URL, nil)
	if err != nil {
		return nil, err
	}

	// Set default headers
	req.Header.Set("User-Agent", "Mango Downloader/1.0")

	// Set referer if provided
	if params.Referer != "" {
		req.Header.Set("Referer", params.Referer)
	}

	// Set additional headers if provided
	for key, value := range params.Headers {
		req.Header.Set(key, value)
	}

	resp, err := Client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, &HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			URL:        params.URL,
		}
	}

	return resp.Body, nil
}

// HTTPError represents an HTTP error
type HTTPError struct {
	StatusCode int
	Status     string
	URL        string
}

func (e *HTTPError) Error() string {
	return "HTTP " + e.Status + " for URL: " + e.URL
}
