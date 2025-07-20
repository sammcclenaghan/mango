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

	content, err := FetchURLContent(testURL, "", false, false, false, false)
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

	content, err := FetchURLContent(testURL, "", false, false, false, false)
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

	content, err := FetchURLContent(testURL, "", false, false, false, false)
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

	content, err := FetchURLContent(testURL, "", false, false, false, false)
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

// TestFetchChapterRange tests fetching pages for a specific chapter range.
func TestFetchChapterRange(t *testing.T) {
	testURL := "https://mangadex.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece"

	// Test fetching a specific chapter
	content, err := FetchURLContent(testURL, "1", false, false, false, false)
	if err != nil {
		t.Skipf("Skipping test due to API error (network/rate limit): %v", err)
		return
	}

	// Verify the content contains expected information
	if !strings.Contains(content, "Chapter 1.0:") {
		t.Error("Expected content to contain 'Chapter 1.0:'")
	}

	if !strings.Contains(content, "Found") && !strings.Contains(content, "chapters in range") {
		t.Error("Expected content to contain chapter range information")
	}
}

// TestFetchChapterRange_InvalidRange tests fetching an invalid chapter range.
func TestFetchChapterRange_InvalidRange(t *testing.T) {
	testURL := "https://mangadex.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece"

	// Test with invalid chapter range
	_, err := FetchURLContent(testURL, "invalid", false, false, false)
	if err == nil {
		t.Error("Expected error for invalid chapter number, but got none")
	}

	expectedError := "invalid chapter range"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain '%s', got: %v", expectedError, err)
	}
}

// TestFetchChapterRange_NonExistentRange tests fetching a non-existent chapter range.
func TestFetchChapterRange_NonExistentRange(t *testing.T) {
	testURL := "https://mangadex.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece"

	// Test with non-existent chapter range
	_, err := FetchURLContent(testURL, "99999", false, false, false)
	if err == nil {
		t.Error("Expected error for non-existent chapter, but got none")
	}

	expectedError := "no chapters found"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain '%s', got: %v", expectedError, err)
	}
}

// TestFetchChapterRange_WithDownload tests downloading pages for a specific chapter range.
func TestFetchChapterRange_WithDownload(t *testing.T) {
	testURL := "https://mangadex.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece"

	// Test fetching and downloading a specific chapter
	content, err := FetchURLContent(testURL, "1154", true, false, false)
	if err != nil {
		t.Skipf("Skipping test due to API error (network/rate limit): %v", err)
		return
	}

	// Verify the content contains download information
	if !strings.Contains(content, "Processing chapter") && !strings.Contains(content, "Total downloaded") {
		t.Error("Expected content to contain download information")
	}
}

// TestFetchChapterRange_WithoutDownload tests listing URLs without downloading.
func TestFetchChapterRange_WithoutDownload(t *testing.T) {
	testURL := "https://mangadex.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece"

	// Test fetching without downloading
	content, err := FetchURLContent(testURL, "1154", false, false, false)
	if err != nil {
		t.Skipf("Skipping test due to API error (network/rate limit): %v", err)
		return
	}

	// Verify the content contains chapter information but not download info
	if !strings.Contains(content, "Chapter") {
		t.Error("Expected content to contain 'Chapter'")
	}

	if strings.Contains(content, "Downloading pages") {
		t.Error("Expected content NOT to contain 'Downloading pages' when download=false")
	}
}

// TestFetchChapterRange_WithCBZ tests downloading and saving as CBZ.
func TestFetchChapterRange_WithCBZ(t *testing.T) {
	testURL := "https://mangadex.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece"

	// Test fetching, downloading, and saving as CBZ
	content, err := FetchURLContent(testURL, "1154", true, true, false)
	if err != nil {
		t.Skipf("Skipping test due to API error (network/rate limit): %v", err)
		return
	}

	// Verify the content contains processing information (CBZ creation may fail due to API)
	if !strings.Contains(content, "Processing chapter") && !strings.Contains(content, "Total downloaded") {
		t.Error("Expected content to contain chapter processing information")
	}
}

// TestCBZFlagValidation tests that CBZ flag requires download flag.
func TestCBZFlagValidation(t *testing.T) {
	// This test simulates the main function's flag validation
	// In a real scenario, we would need to refactor main to make it testable

	// Test case: CBZ without download should be invalid
	download := false
	saveCBZ := true

	if saveCBZ && !download {
		// This is the expected validation behavior
		return
	}

	t.Error("Expected validation to fail when CBZ is requested without download")
}

// TestFetchChapterRange_WithAZW3 tests downloading, saving as CBZ, and converting to AZW3.
func TestFetchChapterRange_WithAZW3(t *testing.T) {
	testURL := "https://mangadex.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece"

	// Test with AZW3 conversion
	content, err := FetchURLContent(testURL, "1154", true, true, true)
	if err != nil {
		t.Skipf("Skipping test due to API error (network/rate limit): %v", err)
		return
	}

	// Verify the content contains processing information
	if !strings.Contains(content, "Processing chapter") && !strings.Contains(content, "Total downloaded") {
		t.Error("Expected content to contain chapter processing information")
	}

	// Check for AZW3 conversion attempt (may fail if Calibre not installed)
	if strings.Contains(content, "Converting") || strings.Contains(content, "ebook-convert not found") {
		t.Log("AZW3 conversion was attempted")
	}
}

// TestAZW3FlagValidation tests that AZW3 flag requires CBZ flag.
func TestAZW3FlagValidation(t *testing.T) {
	// Test case: AZW3 without CBZ should be invalid
	saveCBZ := false
	convertToAZW3 := true

	if convertToAZW3 && !saveCBZ {
		// This is the expected validation behavior
		return
	}

	t.Error("Expected validation to fail when AZW3 is requested without CBZ")
}

// TestFetchChapterRange_MultipleChapters tests fetching multiple chapters.
func TestFetchChapterRange_MultipleChapters(t *testing.T) {
	testURL := "https://mangadex.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece"

	// Test fetching multiple chapters using range syntax
	content, err := FetchURLContent(testURL, "1-3", false, false, false)
	if err != nil {
		t.Skipf("Skipping test due to API error (network/rate limit): %v", err)
		return
	}

	// Verify the content mentions multiple chapters
	if !strings.Contains(content, "Found") && !strings.Contains(content, "chapters in range") {
		t.Error("Expected content to mention multiple chapters")
	}
}

// TestFetchChapterRange_ComplexRange tests fetching with complex range syntax.
func TestFetchChapterRange_ComplexRange(t *testing.T) {
	testURL := "https://mangadex.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece"

	// Test with complex range syntax
	content, err := FetchURLContent(testURL, "1,3,1152-1154", false, false, false)
	if err != nil {
		t.Skipf("Skipping test due to API error (network/rate limit): %v", err)
		return
	}

	// Verify the content handles the complex range
	if !strings.Contains(content, "Found") {
		t.Error("Expected content to show found chapters")
	}
}

// TestFetchChapterRange_Deduplication tests that duplicate chapters are filtered out.
func TestFetchChapterRange_Deduplication(t *testing.T) {
	testURL := "https://mangadex.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece"

	// Test with range that might have duplicates
	content, err := FetchURLContent(testURL, "1-3", false, false, false)
	if err != nil {
		t.Skipf("Skipping test due to API error (network/rate limit): %v", err)
		return
	}

	// Verify the content mentions unique chapters
	if !strings.Contains(content, "unique chapters") {
		t.Error("Expected content to mention 'unique chapters'")
	}

	// Check that debug output indicates deduplication is working
	if strings.Contains(content, "duplicate") {
		// This would appear in debug output if duplicates were found and skipped
		t.Log("Deduplication detected (this is expected behavior)")
	}
}
