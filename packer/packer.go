package packer

import (
	"archive/zip"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.sammcclenaghan.com/mango/downloader"
)

// ProgressCallback is a function type for progress updates during packing
type ProgressCallback func(page, progress int)

// ArchiveCBZ archives the given files into a CBZ file
func ArchiveCBZ(filename string, files []*downloader.File, progress ProgressCallback) error {
	if len(files) == 0 {
		return errors.New("no files to pack")
	}

	// Ensure the filename has .cbz extension
	if !strings.HasSuffix(strings.ToLower(filename), ".cbz") {
		filename += ".cbz"
	}

	// Create the directory if it doesn't exist
	dir := filepath.Dir(filename)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create the CBZ file
	buff, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("file %s already exists", filename)
		}
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer buff.Close()

	w := zip.NewWriter(buff)
	defer w.Close()

	for i, file := range files {
		// Use page number for filename instead of index to maintain order
		filename := fmt.Sprintf("%03d.jpg", file.Page)

		f, err := w.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to create entry %s: %w", filename, err)
		}

		if _, err = f.Write(file.Data); err != nil {
			return fmt.Errorf("failed to write data for %s: %w", filename, err)
		}

		// Report progress
		if progress != nil {
			progress(1, i)
		}
	}

	return nil
}

// GetCBZFilename generates a standardized CBZ filename from manga title and chapter info
func GetCBZFilename(title string, chapterNumber float64, chapterTitle string) string {
	// Sanitize title for filename
	sanitizedTitle := sanitizeFilename(title)

	// Format chapter number
	chapterStr := fmt.Sprintf("%.1f", chapterNumber)
	if chapterNumber == float64(int64(chapterNumber)) {
		chapterStr = fmt.Sprintf("%.0f", chapterNumber)
	}

	// Create base filename
	filename := fmt.Sprintf("%s - Chapter %s", sanitizedTitle, chapterStr)

	// Add chapter title if provided
	if chapterTitle != "" {
		sanitizedChapterTitle := sanitizeFilename(chapterTitle)
		filename += fmt.Sprintf(" - %s", sanitizedChapterTitle)
	}

	return filename + ".cbz"
}

// sanitizeFilename removes or replaces characters that are invalid in filenames
func sanitizeFilename(filename string) string {
	// Replace invalid characters with underscores
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := filename

	for _, char := range invalidChars {
		result = strings.ReplaceAll(result, char, "_")
	}

	// Trim spaces and dots from the end
	result = strings.TrimRight(result, " .")

	// Limit length to avoid filesystem issues
	if len(result) > 200 {
		result = result[:200]
	}

	return result
}

// ArchiveMultipleChapters creates separate CBZ files for multiple chapters
func ArchiveMultipleChapters(baseDir string, chapters map[string][]*downloader.File, titles map[string]string, chapterNumbers map[string]float64, progress ProgressCallback) error {
	if len(chapters) == 0 {
		return errors.New("no chapters to pack")
	}

	// Create base directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create base directory %s: %w", baseDir, err)
	}

	totalFiles := 0
	for _, files := range chapters {
		totalFiles += len(files)
	}

	processedFiles := 0
	for chapterKey, files := range chapters {
		if len(files) == 0 {
			continue
		}

		title := titles[chapterKey]
		chapterNum := chapterNumbers[chapterKey]

		filename := GetCBZFilename(title, chapterNum, "")
		fullPath := filepath.Join(baseDir, filename)

		chapterProgress := func(page, fileProgress int) {
			if progress != nil {
				progress(page, processedFiles+fileProgress)
			}
		}

		if err := ArchiveCBZ(fullPath, files, chapterProgress); err != nil {
			return fmt.Errorf("failed to archive chapter %s: %w", chapterKey, err)
		}

		processedFiles += len(files)
	}

	return nil
}

// BundleChapters combines multiple chapters into a single CBZ file
func BundleChapters(filename string, chapters map[string][]*downloader.File, progress ProgressCallback) error {
	if len(chapters) == 0 {
		return errors.New("no chapters to bundle")
	}

	// Collect all files with chapter prefixes
	var allFiles []*downloader.File

	for _, files := range chapters {
		for _, file := range files {
			// Create a new file with modified page numbering for bundling
			bundledFile := &downloader.File{
				Data: file.Data,
				Page: file.Page, // Keep original page number, we'll handle ordering in filename
			}
			allFiles = append(allFiles, bundledFile)
		}
	}

	if len(allFiles) == 0 {
		return errors.New("no files to bundle")
	}

	return ArchiveCBZ(filename, allFiles, progress)
}
