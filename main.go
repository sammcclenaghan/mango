package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

// FetchURLContent fetches the content from the given URL and returns it as a string.
func FetchURLContent(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(body), nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: mango <url>")
		return
	}

	url := os.Args[1]
	content, err := FetchURLContent(url)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(content)
}
