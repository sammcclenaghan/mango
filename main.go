package main

import (
	"fmt"
	"os"

	"github.sammcclenaghan.com/mango/colors"
	"github.sammcclenaghan.com/mango/converter"
	"github.sammcclenaghan.com/mango/downloader"
	"github.sammcclenaghan.com/mango/grabber"
	"github.sammcclenaghan.com/mango/packer"
	"github.sammcclenaghan.com/mango/ranges"
)

// FetchURLContent fetches the content from the given URL and returns it as a string.
func FetchURLContent(url string, chapterRange string, download bool, saveCBZ bool, convertToAZW3 bool) (string, error) {
	// Create a base grabber
	g := &grabber.Grabber{
		URL: url,
		Settings: grabber.Settings{
			Language: "en", // default to English
		},
	}

	// Create MangaDx grabber
	mangadx := grabber.NewMangadx(g)

	// Test if this is a supported site
	isSupported, err := mangadx.Test()
	if err != nil {
		return "", fmt.Errorf("error testing site: %w", err)
	}

	if !isSupported {
		return "", fmt.Errorf("unsupported site: %s", url)
	}

	// Fetch the title
	title, err := mangadx.FetchTitle()
	if err != nil {
		return "", fmt.Errorf("error fetching title: %w", err)
	}

	// Fetch chapters
	chapters, errs := mangadx.FetchChapters()
	if len(errs) > 0 {
		return "", fmt.Errorf("errors fetching chapters: %v", errs)
	}

	// Build output string
	output := fmt.Sprintf("Title: %s\n", title)
	output += fmt.Sprintf("Found %d chapters:\n\n", len(chapters))

	// If a specific chapter range is requested, fetch those chapters
	if chapterRange != "" {
		colors.DebugPrintf("Debug: Looking for chapter range %s\n", chapterRange)
		colors.DebugPrintf("Debug: Available chapters: %d\n", len(chapters))
		return fetchChapterRange(mangadx, chapters, chapterRange, title, download, saveCBZ, convertToAZW3)
	}

	// Otherwise, list all chapters
	for _, chapter := range chapters {
		output += fmt.Sprintf("Chapter %.1f: %s (%s)\n",
			chapter.GetNumber(),
			chapter.GetTitle(),
			chapter.GetLanguage())
	}

	return output, nil
}

// fetchChapterRange fetches pages for chapters within the specified range
func fetchChapterRange(mangadx *grabber.Mangadx, chapters grabber.Filterables, chapterRange string, title string, download bool, saveCBZ bool, convertToAZW3 bool) (string, error) {
	// Parse the chapter range
	parsedRanges, err := ranges.Parse(chapterRange)
	if err != nil {
		return "", fmt.Errorf("invalid chapter range '%s': %w", chapterRange, err)
	}

	// Find matching chapters and deduplicate by chapter number
	var selectedChapters []grabber.Filterable
	seenChapters := make(map[float64]bool)
	duplicateCount := 0
	for _, chapter := range chapters {
		if ranges.ContainsAny(parsedRanges, chapter.GetNumber()) {
			// Only add if we haven't seen this chapter number before
			if !seenChapters[chapter.GetNumber()] {
				selectedChapters = append(selectedChapters, chapter)
				seenChapters[chapter.GetNumber()] = true
				colors.FetchedPrintf("fetching %s chapter %.0f\n", title, chapter.GetNumber())
			} else {
				duplicateCount++
				colors.DebugPrintf("Debug: Skipping duplicate chapter %.1f (%s)\n", chapter.GetNumber(), chapter.GetLanguage())
			}
		}
	}

	if duplicateCount > 0 {
		colors.DebugPrintf("Debug: Skipped %d duplicate chapters\n", duplicateCount)
	}

	if len(selectedChapters) == 0 {
		// List available chapters for debugging
		availableChapters := ""
		for _, ch := range chapters {
			availableChapters += fmt.Sprintf("%.1f ", ch.GetNumber())
		}
		return "", fmt.Errorf("no chapters found for range %s. Available chapters: %s", chapterRange, availableChapters)
	}

	// Build initial output
	output := fmt.Sprintf("Title: %s\n", title)
	output += fmt.Sprintf("Found %d unique chapters in range %s:\n\n", len(selectedChapters), chapterRange)

	if !download {
		// Just list the matching chapters
		for _, chapter := range selectedChapters {
			output += fmt.Sprintf("Chapter %.1f: %s (%s)\n",
				chapter.GetNumber(),
				chapter.GetTitle(),
				chapter.GetLanguage())
		}
		return output, nil
	}

	// Download mode - process each chapter
	var allFiles []*downloader.File
	var downloadedChapters []*grabber.Chapter

	for _, selectedChapter := range selectedChapters {
		colors.FetchedPrintf("fetching %s chapter %.0f\n", title, selectedChapter.GetNumber())

		// Debug: Print chapter ID before fetching
		if mangadxChap, ok := selectedChapter.(*grabber.MangadxChapter); ok {
			colors.DebugPrintf("Debug: Fetching chapter ID: %s\n", mangadxChap.Id)
		}

		// Fetch the chapter with its pages
		chapterWithPages, err := mangadx.FetchChapter(selectedChapter)
		if err != nil {
			colors.ErrorPrintf("❌ Error fetching chapter %.1f: %v\n", selectedChapter.GetNumber(), err)
			continue
		}

		downloadedChapters = append(downloadedChapters, chapterWithPages)

		// Download the chapter pages
		colors.DownloadedPrintf("downloading %s chapter %.0f\n", title, chapterWithPages.Number)
		progressCallback := func(page, progress int, err error) {
			if err != nil {
				colors.ErrorPrintf("Error downloading page %d: %v\n", page, err)
			}
		}

		files, err := downloader.FetchChapter(mangadx, chapterWithPages, progressCallback)
		if err != nil {
			colors.ErrorPrintf("Error downloading chapter %.1f: %v\n", chapterWithPages.Number, err)
			continue
		}

		allFiles = append(allFiles, files...)
		colors.SavedPrintf("saving %s chapter %.0f\n", title, chapterWithPages.Number)
	}

	output += fmt.Sprintf("\nTotal downloaded: %d pages from %d chapters\n", len(allFiles), len(downloadedChapters))

	// Save to CBZ if requested
	if saveCBZ && len(allFiles) > 0 {
		if len(downloadedChapters) == 1 {
			// Single chapter - use normal filename
			chapter := downloadedChapters[0]
			cbzFilename := packer.GetCBZFilename(title, chapter.Number, chapter.Title)
			colors.SavedPrintf("saving to cbz\n")

			packingCallback := func(page, progress int) {
				// Silent packing
			}

			err := packer.ArchiveCBZ(cbzFilename, allFiles, packingCallback)
			if err != nil {
				return "", fmt.Errorf("error creating CBZ file: %w", err)
			}

			output += fmt.Sprintf("Successfully created CBZ file: %s\n", cbzFilename)

			// Convert to AZW3 if requested
			if convertToAZW3 {
				output += performAZW3Conversion(cbzFilename)
			}
		} else {
			// Multiple chapters - bundle them
			bundleFilename := packer.GetCBZFilename(title, 0, fmt.Sprintf("Chapters %s", chapterRange))
			colors.SavedPrintf("saving to cbz\n")

			packingCallback := func(page, progress int) {
				// Silent packing
			}

			err := packer.ArchiveCBZ(bundleFilename, allFiles, packingCallback)
			if err != nil {
				return "", fmt.Errorf("error creating bundled CBZ file: %w", err)
			}

			output += fmt.Sprintf("Successfully created bundled CBZ file: %s\n", bundleFilename)

			// Convert to AZW3 if requested
			if convertToAZW3 {
				output += performAZW3Conversion(bundleFilename)
			}
		}
	} else if !saveCBZ {
		// List downloaded file information
		chapterFileCount := make(map[float64]int)
		for _, file := range allFiles {
			// This is a simplified approach - in a real implementation,
			// we'd need to track which files belong to which chapter
			chapterFileCount[0] += len(file.Data)
		}
		output += fmt.Sprintf("Total downloaded data: %d bytes\n", chapterFileCount[0])
	}

	return output, nil
}

// performAZW3Conversion converts a CBZ file to AZW3 format
func performAZW3Conversion(cbzFile string) string {
	output := ""

	// Check if ebook-convert is available
	if !converter.IsEbookConvertAvailable() {
		output += colors.Warning("Warning: ebook-convert not found. Please install Calibre to enable AZW3 conversion.\n")
		output += "Download from: https://calibre-ebook.com/download\n"
		return output
	}

	conv := converter.NewConverter()
	conv.DeleteSource = false // Keep CBZ file by default

	output += fmt.Sprintf("Converting %s to AZW3 format...\n", cbzFile)

	// Generate output filename
	azw3File := conv.GenerateOutputPath(cbzFile, ".azw3")

	result, err := conv.ConvertCBZToAZW3(cbzFile, azw3File)
	if err != nil {
		output += colors.Error(fmt.Sprintf("❌ Error converting to AZW3: %v\n", err))
		return output
	}

	if result.Success {
		output += fmt.Sprintf("Successfully converted to AZW3: %s (%d bytes)\n", result.OutputFile, result.BytesWritten)
	} else {
		output += colors.Error(fmt.Sprintf("AZW3 conversion failed: %v\n", result.Error))
	}

	return output
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: mango <url> [chapter_range] [--download] [--cbz] [--azw3]")
		fmt.Println("Example: mango https://mangadex.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece")
		fmt.Println("Example: mango https://mangadex.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece 1")
		fmt.Println("Example: mango https://mangadex.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece 1-5")
		fmt.Println("Example: mango https://mangadex.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece 1,3,5-10")
		fmt.Println("Example: mango https://mangadex.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece 1-3 --download --cbz")
		fmt.Println("Example: mango https://mangadex.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece 1-3 --download --cbz --azw3")
		fmt.Println("Use --download flag to actually download pages (default: just list URLs)")
		fmt.Println("Use --cbz flag to save downloaded pages as CBZ file (requires --download)")
		fmt.Println("Use --azw3 flag to convert CBZ to AZW3 format for Kindle (requires --cbz and Calibre)")
		return
	}

	url := os.Args[1]
	var chapterRange string
	download := false
	saveCBZ := false
	convertToAZW3 := false

	// Parse remaining arguments
	for i := 2; i < len(os.Args); i++ {
		arg := os.Args[i]
		if arg == "--download" {
			download = true
		} else if arg == "--cbz" {
			saveCBZ = true
		} else if arg == "--azw3" {
			convertToAZW3 = true
		} else if chapterRange == "" {
			chapterRange = arg
		}
	}

	// Validate flags
	if saveCBZ && !download {
		colors.ErrorPrintf("Error: --cbz flag requires --download flag\n")
		return
	}

	if convertToAZW3 && !saveCBZ {
		colors.ErrorPrintf("Error: --azw3 flag requires --cbz flag\n")
		return
	}

	content, err := FetchURLContent(url, chapterRange, download, saveCBZ, convertToAZW3)
	if err != nil {
		colors.ErrorPrintf("Error: %v\n", err)
		return
	}

	fmt.Println(content)
}
