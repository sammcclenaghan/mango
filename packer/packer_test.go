package packer

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.sammcclenaghan.com/mango/downloader"
)

func TestArchiveCBZ_Success(t *testing.T) {
	// Create test files
	files := []*downloader.File{
		{Data: []byte("page 1 data"), Page: 1},
		{Data: []byte("page 2 data"), Page: 2},
		{Data: []byte("page 3 data"), Page: 3},
	}

	// Create temporary directory
	tempDir := t.TempDir()
	filename := filepath.Join(tempDir, "test.cbz")

	// Track progress
	var progressCalls []int
	progressCallback := func(page, progress int) {
		progressCalls = append(progressCalls, progress)
	}

	// Test archiving
	err := ArchiveCBZ(filename, files, progressCallback)
	if err != nil {
		t.Fatalf("ArchiveCBZ() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Error("CBZ file was not created")
	}

	// Verify ZIP contents
	reader, err := zip.OpenReader(filename)
	if err != nil {
		t.Fatalf("Failed to open CBZ file: %v", err)
	}
	defer reader.Close()

	if len(reader.File) != 3 {
		t.Errorf("Expected 3 files in CBZ, got %d", len(reader.File))
	}

	// Verify file contents
	for i, zipFile := range reader.File {
		expectedName := "001.jpg"
		if i == 1 {
			expectedName = "002.jpg"
		} else if i == 2 {
			expectedName = "003.jpg"
		}

		if zipFile.Name != expectedName {
			t.Errorf("Expected file name %s, got %s", expectedName, zipFile.Name)
		}

		rc, err := zipFile.Open()
		if err != nil {
			t.Fatalf("Failed to open file in ZIP: %v", err)
		}

		var buf bytes.Buffer
		_, err = buf.ReadFrom(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("Failed to read file content: %v", err)
		}

		expectedData := files[i].Data
		if !bytes.Equal(buf.Bytes(), expectedData) {
			t.Errorf("File %d content mismatch", i)
		}
	}

	// Verify progress was called
	if len(progressCalls) != 3 {
		t.Errorf("Expected 3 progress calls, got %d", len(progressCalls))
	}
}

func TestArchiveCBZ_EmptyFiles(t *testing.T) {
	tempDir := t.TempDir()
	filename := filepath.Join(tempDir, "empty.cbz")

	err := ArchiveCBZ(filename, []*downloader.File{}, nil)
	if err == nil {
		t.Error("Expected error for empty files, but got none")
	}

	expectedError := "no files to pack"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain '%s', got: %v", expectedError, err)
	}
}

func TestArchiveCBZ_FileExists(t *testing.T) {
	tempDir := t.TempDir()
	filename := filepath.Join(tempDir, "existing.cbz")

	// Create existing file
	_, err := os.Create(filename)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	files := []*downloader.File{
		{Data: []byte("test data"), Page: 1},
	}

	err = ArchiveCBZ(filename, files, nil)
	if err == nil {
		t.Error("Expected error for existing file, but got none")
	}

	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("Expected error about existing file, got: %v", err)
	}
}

func TestArchiveCBZ_AddsCBZExtension(t *testing.T) {
	tempDir := t.TempDir()
	filename := filepath.Join(tempDir, "test")

	files := []*downloader.File{
		{Data: []byte("test data"), Page: 1},
	}

	err := ArchiveCBZ(filename, files, nil)
	if err != nil {
		t.Fatalf("ArchiveCBZ() error = %v", err)
	}

	// Verify file was created with .cbz extension
	expectedFilename := filename + ".cbz"
	if _, err := os.Stat(expectedFilename); os.IsNotExist(err) {
		t.Error("CBZ file with .cbz extension was not created")
	}
}

func TestGetCBZFilename(t *testing.T) {
	tests := []struct {
		name           string
		title          string
		chapterNumber  float64
		chapterTitle   string
		expectedPrefix string
	}{
		{
			name:           "basic filename",
			title:          "One Piece",
			chapterNumber:  1,
			chapterTitle:   "Romance Dawn",
			expectedPrefix: "One Piece - Chapter 1 - Romance Dawn.cbz",
		},
		{
			name:           "decimal chapter number",
			title:          "Test Manga",
			chapterNumber:  1.5,
			chapterTitle:   "Half Chapter",
			expectedPrefix: "Test Manga - Chapter 1.5 - Half Chapter.cbz",
		},
		{
			name:           "no chapter title",
			title:          "Simple Manga",
			chapterNumber:  10,
			chapterTitle:   "",
			expectedPrefix: "Simple Manga - Chapter 10.cbz",
		},
		{
			name:           "title with invalid characters",
			title:          "Manga: Test/Title",
			chapterNumber:  1,
			chapterTitle:   "Chapter*Title",
			expectedPrefix: "Manga_ Test_Title - Chapter 1 - Chapter_Title.cbz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetCBZFilename(tt.title, tt.chapterNumber, tt.chapterTitle)
			if result != tt.expectedPrefix {
				t.Errorf("GetCBZFilename() = %v, want %v", result, tt.expectedPrefix)
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic string",
			input:    "Normal Title",
			expected: "Normal Title",
		},
		{
			name:     "with invalid characters",
			input:    "Title/with\\invalid:chars*?\"<>|",
			expected: "Title_with_invalid_chars______",
		},
		{
			name:     "with trailing spaces and dots",
			input:    "Title with spaces... ",
			expected: "Title with spaces",
		},
		{
			name:     "very long title",
			input:    strings.Repeat("A", 250),
			expected: strings.Repeat("A", 200),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFilename() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestArchiveMultipleChapters(t *testing.T) {
	tempDir := t.TempDir()

	// Create test data
	chapters := map[string][]*downloader.File{
		"chapter1": {
			{Data: []byte("ch1 page1"), Page: 1},
			{Data: []byte("ch1 page2"), Page: 2},
		},
		"chapter2": {
			{Data: []byte("ch2 page1"), Page: 1},
		},
	}

	titles := map[string]string{
		"chapter1": "Test Manga",
		"chapter2": "Test Manga",
	}

	chapterNumbers := map[string]float64{
		"chapter1": 1,
		"chapter2": 2,
	}

	var progressCalls []int
	progressCallback := func(page, progress int) {
		progressCalls = append(progressCalls, progress)
	}

	err := ArchiveMultipleChapters(tempDir, chapters, titles, chapterNumbers, progressCallback)
	if err != nil {
		t.Fatalf("ArchiveMultipleChapters() error = %v", err)
	}

	// Verify files were created
	expectedFiles := []string{
		"Test Manga - Chapter 1.cbz",
		"Test Manga - Chapter 2.cbz",
	}

	for _, expectedFile := range expectedFiles {
		fullPath := filepath.Join(tempDir, expectedFile)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("Expected file %s was not created", expectedFile)
		}
	}

	// Verify progress was called
	if len(progressCalls) < 2 {
		t.Errorf("Expected at least 2 progress calls, got %d", len(progressCalls))
	}
}

func TestBundleChapters(t *testing.T) {
	tempDir := t.TempDir()
	filename := filepath.Join(tempDir, "bundled.cbz")

	chapters := map[string][]*downloader.File{
		"chapter1": {
			{Data: []byte("ch1 page1"), Page: 1},
			{Data: []byte("ch1 page2"), Page: 2},
		},
		"chapter2": {
			{Data: []byte("ch2 page1"), Page: 1},
		},
	}

	var progressCalls []int
	progressCallback := func(page, progress int) {
		progressCalls = append(progressCalls, progress)
	}

	err := BundleChapters(filename, chapters, progressCallback)
	if err != nil {
		t.Fatalf("BundleChapters() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Error("Bundled CBZ file was not created")
	}

	// Verify ZIP contents
	reader, err := zip.OpenReader(filename)
	if err != nil {
		t.Fatalf("Failed to open bundled CBZ file: %v", err)
	}
	defer reader.Close()

	// Should contain all files from all chapters
	expectedFileCount := 3
	if len(reader.File) != expectedFileCount {
		t.Errorf("Expected %d files in bundled CBZ, got %d", expectedFileCount, len(reader.File))
	}
}

func TestBundleChapters_EmptyChapters(t *testing.T) {
	tempDir := t.TempDir()
	filename := filepath.Join(tempDir, "empty_bundle.cbz")

	err := BundleChapters(filename, map[string][]*downloader.File{}, nil)
	if err == nil {
		t.Error("Expected error for empty chapters, but got none")
	}

	expectedError := "no chapters to bundle"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain '%s', got: %v", expectedError, err)
	}
}

func TestArchiveCBZ_CreatesDirectory(t *testing.T) {
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "subdir", "nested")
	filename := filepath.Join(subDir, "test.cbz")

	files := []*downloader.File{
		{Data: []byte("test data"), Page: 1},
	}

	err := ArchiveCBZ(filename, files, nil)
	if err != nil {
		t.Fatalf("ArchiveCBZ() error = %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(subDir); os.IsNotExist(err) {
		t.Error("Expected directory was not created")
	}

	// Verify file was created
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Error("CBZ file was not created")
	}
}
