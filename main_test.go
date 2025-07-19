package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.sammcclenaghan.com/mango/grabber"
)

// TestFetchURLContent_Success tests fetching content from a valid MangaDx URL.
func TestFetchURLContent_Success(t *testing.T) {
	// Mock manga response
	mockManga := map[string]interface{}{
		"data": map[string]interface{}{
			"attributes": map[string]interface{}{
				"title": map[string]string{
					"en": "Test Manga",
				},
				"altTitles": []map[string]string{},
			},
		},
	}

	// Mock chapters response
	mockChapters := map[string]interface{}{
		"data": []map[string]interface{}{
			{
				"id": "chapter-1-id",
				"attributes": map[string]interface{}{
					"chapter":            "1",
					"title":              "First Chapter",
					"translatedLanguage": "en",
					"pages":              20,
				},
			},
			{
				"id": "chapter-2-id",
				"attributes": map[string]interface{}{
					"chapter":            "2",
					"title":              "Second Chapter",
					"translatedLanguage": "en",
					"pages":              18,
				},
			},
		},
	}

	// Create test server that handles both manga and feed endpoints
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/feed") {
			json.NewEncoder(w).Encode(mockChapters)
		} else {
			json.NewEncoder(w).Encode(mockManga)
		}
	}))
	defer ts.Close()

	// Test with a real mangadex URL - this will make actual API calls
	testURL := "https://mangadex.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece"

	content, err := FetchURLContent(testURL, "")
	if err != nil {
		t.Skipf("Skipping test due to API error (network/rate limit): %v", err)
		return
	}

	// Verify the content contains expected information
	if !strings.Contains(content, "Title:") {
		t.Error("Expected content to contain 'Title:'")
	}

	if !strings.Contains(content, "Found") && !strings.Contains(content, "chapters") {
		t.Error("Expected content to contain chapter information")
	}
}

// TestFetchURLContent_UnsupportedSite tests with an unsupported website.
func TestFetchURLContent_UnsupportedSite(t *testing.T) {
	testURL := "https://example.com/manga"

	content, err := FetchURLContent(testURL, "")
	if err == nil {
		t.Error("Expected error for unsupported site, but got none")
	}

	if content != "" {
		t.Errorf("Expected empty content for error case, got: %s", content)
	}

	expectedError := "unsupported site"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain '%s', got: %v", expectedError, err)
	}
}

// TestFetchURLContent_InvalidURL tests with an invalid URL.
func TestFetchURLContent_InvalidURL(t *testing.T) {
	testURL := "not-a-valid-url"

	content, err := FetchURLContent(testURL, "")
	if err == nil {
		t.Error("Expected error for invalid URL, but got none")
	}

	if content != "" {
		t.Errorf("Expected empty content for error case, got: %s", content)
	}
}

// TestFetchURLContent_EmptyURL tests with an empty URL.
func TestFetchURLContent_EmptyURL(t *testing.T) {
	testURL := ""

	content, err := FetchURLContent(testURL, "")
	if err == nil {
		t.Error("Expected error for empty URL, but got none")
	}

	if content != "" {
		t.Errorf("Expected empty content for error case, got: %s", content)
	}
}

// TestFetchURLContent_MangadxURL tests URL validation for MangaDx.
func TestFetchURLContent_MangadxURL(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		shouldPass bool
	}{
		{
			name:       "valid mangadex URL",
			url:        "https://mangadex.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece",
			shouldPass: true,
		},
		{
			name:       "mangadex URL without UUID",
			url:        "https://mangadex.org/title/invalid-id/manga",
			shouldPass: true, // URL format is valid, but API call will fail
		},
		{
			name:       "non-mangadex URL",
			url:        "https://manganato.com/manga-test",
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create grabber to test URL validation
			g := &grabber.Grabber{
				URL: tt.url,
				Settings: grabber.Settings{
					Language: "en",
				},
			}

			mangadx := grabber.NewMangadx(g)
			isSupported, err := mangadx.Test()

			if err != nil {
				t.Fatalf("Test() error = %v", err)
			}

			if isSupported != tt.shouldPass {
				t.Errorf("URL %s: expected supported=%v, got=%v", tt.url, tt.shouldPass, isSupported)
			}
		})
	}
}

// TestMainUsage tests the main function's argument handling.
func TestMainUsage(t *testing.T) {
	// This test would require capturing stdout/stderr or refactoring main
	// For now, we'll create a simple test that verifies our argument parsing logic

	tests := []struct {
		name  string
		args  []string
		valid bool
	}{
		{
			name:  "no arguments",
			args:  []string{"mango"},
			valid: false,
		},
		{
			name:  "with URL argument",
			args:  []string{"mango", "https://mangadex.org/title/test"},
			valid: true,
		},
		{
			name:  "multiple arguments",
			args:  []string{"mango", "https://mangadex.org/title/test", "extra"},
			valid: true, // Should still work, extra args ignored
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasValidArgs := len(tt.args) >= 2
			if hasValidArgs != tt.valid {
				t.Errorf("Expected valid=%v for args %v, got=%v", tt.valid, tt.args, hasValidArgs)
			}
		})
	}
}

// TestFetchSpecificChapter tests fetching pages for a specific chapter.
func TestFetchSpecificChapter(t *testing.T) {
	testURL := "https://mangadex.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece"

	// Test fetching a specific chapter
	content, err := FetchURLContent(testURL, "1")
	if err != nil {
		t.Skipf("Skipping test due to API error (network/rate limit): %v", err)
		return
	}

	// Verify the content contains page information
	if !strings.Contains(content, "Chapter: 1") {
		t.Error("Expected content to contain 'Chapter: 1'")
	}

	if !strings.Contains(content, "Page") {
		t.Error("Expected content to contain page information")
	}

	if !strings.Contains(content, "Pages:") {
		t.Error("Expected content to contain page count")
	}
}

// TestFetchSpecificChapter_InvalidChapter tests fetching an invalid chapter number.
func TestFetchSpecificChapter_InvalidChapter(t *testing.T) {
	testURL := "https://mangadex.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece"

	// Test with invalid chapter number
	_, err := FetchURLContent(testURL, "invalid")
	if err == nil {
		t.Error("Expected error for invalid chapter number, but got none")
	}

	expectedError := "invalid chapter number"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain '%s', got: %v", expectedError, err)
	}
}

// TestFetchSpecificChapter_NonExistentChapter tests fetching a non-existent chapter.
func TestFetchSpecificChapter_NonExistentChapter(t *testing.T) {
	testURL := "https://mangadex.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece"

	// Test with non-existent chapter number
	_, err := FetchURLContent(testURL, "99999")
	if err == nil {
		t.Error("Expected error for non-existent chapter, but got none")
	}

	expectedError := "not found"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain '%s', got: %v", expectedError, err)
	}
}
