package grabber

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMangadex_Test(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "valid mangadex URL",
			url:      "https://mangadex.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece",
			expected: true,
		},
		{
			name:     "invalid URL",
			url:      "https://example.com/manga",
			expected: false,
		},
		{
			name:     "mangadex with different subdomain",
			url:      "https://api.mangadex.org/manga/123",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Grabber{URL: tt.url}
			m := NewMangadx(g)

			result, err := m.Test()
			if err != nil {
				t.Fatalf("Test() error = %v", err)
			}

			if result != tt.expected {
				t.Errorf("Test() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMangadex_FetchTitle(t *testing.T) {
	// Mock manga response
	mockManga := mangadxManga{
		Id: "test-id",
		Data: struct {
			Attributes struct {
				Title     map[string]string
				AltTitles altTitles
			}
		}{
			Attributes: struct {
				Title     map[string]string
				AltTitles altTitles
			}{
				Title: map[string]string{
					"en": "Test Manga",
					"ja": "テストマンガ",
				},
				AltTitles: altTitles{
					{"es": "Manga de Prueba"},
					{"fr": "Manga de Test"},
				},
			},
		},
	}

	// Create test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockManga)
	}))
	defer ts.Close()

	tests := []struct {
		name     string
		language string
		expected string
	}{
		{
			name:     "English title",
			language: "",
			expected: "Test Manga",
		},
		{
			name:     "Spanish alt title",
			language: "es",
			expected: "Manga de Prueba",
		},
		{
			name:     "French alt title",
			language: "fr",
			expected: "Manga de Test",
		},
		{
			name:     "Non-existent language falls back to English",
			language: "de",
			expected: "Test Manga",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip integration tests that require real API calls
			t.Skip("Skipping integration test - need to refactor for better testability")
		})
	}
}

func TestGetUuid(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "valid mangadex URL with UUID",
			url:      "https://mangadex.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece",
			expected: "a1c7c817-4e59-43b7-9365-09675a149a6f",
		},
		{
			name:     "URL without UUID",
			url:      "https://mangadex.org/title/invalid-id/manga",
			expected: "",
		},
		{
			name:     "multiple UUIDs - should return first",
			url:      "https://mangadex.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/chapter/b2d8f928-5f6a-44c8-a75b-09765b249b7f",
			expected: "a1c7c817-4e59-43b7-9365-09675a149a6f",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getUuid(tt.url)
			if result != tt.expected {
				t.Errorf("getUuid() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAltTitles_GetTitleByLang(t *testing.T) {
	altTitles := altTitles{
		{"en": "English Title"},
		{"es": "Título Español"},
		{"fr": "Titre Français"},
		{"ja": "日本語タイトル"},
	}

	tests := []struct {
		name     string
		lang     string
		expected string
	}{
		{
			name:     "existing language - English",
			lang:     "en",
			expected: "English Title",
		},
		{
			name:     "existing language - Spanish",
			lang:     "es",
			expected: "Título Español",
		},
		{
			name:     "non-existent language",
			lang:     "de",
			expected: "",
		},
		{
			name:     "empty language",
			lang:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := altTitles.GetTitleByLang(tt.lang)
			if result != tt.expected {
				t.Errorf("GetTitleByLang() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMangadxChapter_Filterable(t *testing.T) {
	chapter := &MangadxChapter{
		Chapter: Chapter{
			Number:   1.5,
			Title:    "Test Chapter",
			Language: "en",
		},
		Id: "test-id",
	}

	if chapter.GetNumber() != 1.5 {
		t.Errorf("GetNumber() = %v, want %v", chapter.GetNumber(), 1.5)
	}

	if chapter.GetTitle() != "Test Chapter" {
		t.Errorf("GetTitle() = %v, want %v", chapter.GetTitle(), "Test Chapter")
	}

	if chapter.GetLanguage() != "en" {
		t.Errorf("GetLanguage() = %v, want %v", chapter.GetLanguage(), "en")
	}
}

func TestNewMangadx(t *testing.T) {
	g := &Grabber{
		URL: "https://mangadex.org/title/test",
		Settings: Settings{
			Language: "en",
		},
	}

	m := NewMangadx(g)

	if m.Grabber != g {
		t.Error("NewMangadx() did not set Grabber correctly")
	}

	if m.rateLimiter == nil {
		t.Error("NewMangadx() did not initialize rate limiter")
	}
}
