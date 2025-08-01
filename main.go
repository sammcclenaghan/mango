package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.sammcclenaghan.com/mango/colors"
	"github.sammcclenaghan.com/mango/converter"
	"github.sammcclenaghan.com/mango/downloader"
	"github.sammcclenaghan.com/mango/grabber"
	"github.sammcclenaghan.com/mango/packer"
	"github.sammcclenaghan.com/mango/ranges"
)

// FetchURLContent fetches the content from the given URL and returns it as a string.
func FetchURLContent(url string, chapterRange string, download bool, saveCBZ bool, convertToAZW3 bool, convertToEPUB bool, outputDir string, listOnly bool) (string, error) {
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
	if listOnly {
		return listAvailableChapters(title, chapters)
	}

	if chapterRange != "" {
		colors.DebugPrintf("Debug: Looking for chapter range %s\n", chapterRange)
		colors.DebugPrintf("Debug: Available chapters: %d\n", len(chapters))
		return fetchChapterRange(mangadx, chapters, chapterRange, title, download, saveCBZ, convertToAZW3, convertToEPUB, outputDir)
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
func fetchChapterRange(mangadx *grabber.Mangadx, chapters grabber.Filterables, chapterRange string, title string, download bool, saveCBZ bool, convertToAZW3 bool, convertToEPUB bool, outputDir string) (string, error) {
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
		// Create a more helpful error message with suggestions
		var availableNumbers []float64
		for _, ch := range chapters {
			availableNumbers = append(availableNumbers, ch.GetNumber())
		}

		// Sort chapters for better display
		for i := 0; i < len(availableNumbers); i++ {
			for j := i + 1; j < len(availableNumbers); j++ {
				if availableNumbers[i] > availableNumbers[j] {
					availableNumbers[i], availableNumbers[j] = availableNumbers[j], availableNumbers[i]
				}
			}
		}

		// Build available chapters string (limit to first 20 for readability)
		availableStr := ""
		displayCount := len(availableNumbers)
		if displayCount > 20 {
			displayCount = 20
		}

		for i := 0; i < displayCount; i++ {
			if availableNumbers[i] == float64(int64(availableNumbers[i])) {
				availableStr += fmt.Sprintf("%.0f ", availableNumbers[i])
			} else {
				availableStr += fmt.Sprintf("%.1f ", availableNumbers[i])
			}
		}

		if len(availableNumbers) > 20 {
			availableStr += "... (and more)"
		}

		// Suggest some ranges based on available chapters
		suggestions := ""
		if len(availableNumbers) > 0 {
			first := availableNumbers[0]
			if len(availableNumbers) >= 3 {
				third := availableNumbers[2]
				if first == float64(int64(first)) && third == float64(int64(third)) {
					suggestions = fmt.Sprintf("\nTry: %.0f-%.0f or %.0f", first, third, first)
				} else {
					suggestions = fmt.Sprintf("\nTry: %.1f-%.1f or %.1f", first, third, first)
				}
			} else {
				if first == float64(int64(first)) {
					suggestions = fmt.Sprintf("\nTry: %.0f", first)
				} else {
					suggestions = fmt.Sprintf("\nTry: %.1f", first)
				}
			}
		}

		return "", fmt.Errorf("no chapters found for range %s.\nAvailable chapters: %s%s", chapterRange, availableStr, suggestions)
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
	chapterFiles := make(map[float64][]*downloader.File) // Track files by chapter number

	for _, selectedChapter := range selectedChapters {
		colors.FetchedPrintf("fetching %s chapter %.0f\n", title, selectedChapter.GetNumber())

		// Debug: Print chapter ID before fetching
		if mangadxChap, ok := selectedChapter.(*grabber.MangadxChapter); ok {
			colors.DebugPrintf("Debug: Fetching chapter ID: %s\n", mangadxChap.Id)
		}

		// Fetch the chapter with its pages
		chapterWithPages, err := mangadx.FetchChapter(selectedChapter)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				colors.ErrorPrintf("Chapter %.0f not available (404 - possibly licensed/removed)\n", selectedChapter.GetNumber())
			} else {
				colors.ErrorPrintf("Error fetching chapter %.0f: %v\n", selectedChapter.GetNumber(), err)
			}
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
			if strings.Contains(err.Error(), "404") {
				colors.ErrorPrintf("Chapter %.0f pages not available (404 - possibly licensed/removed)\n", chapterWithPages.Number)
			} else {
				colors.ErrorPrintf("Error downloading chapter %.0f: %v\n", chapterWithPages.Number, err)
			}
			continue
		}

		// Store files by chapter number for proper organization
		chapterFiles[chapterWithPages.Number] = files
		allFiles = append(allFiles, files...)
		colors.SavedPrintf("saving %s chapter %.0f\n", title, chapterWithPages.Number)
	}

	if len(downloadedChapters) == 0 {
		return "", fmt.Errorf("no chapters could be downloaded.\n\nThis manga may be:\n• Officially licensed and removed from MangaDx\n• Restricted in your region\n• Temporarily unavailable\n\nSuggestions:\n• Try a different manga series\n• Check official sources like Viz, Crunchyroll, or publisher websites\n• Use --list to verify available chapters")
	}

	output += fmt.Sprintf("\nTotal downloaded: %d pages from %d chapters\n", len(allFiles), len(downloadedChapters))

	// Save to CBZ if requested
	if saveCBZ && len(allFiles) > 0 {
		if len(downloadedChapters) == 1 {
			// Single chapter - use normal filename
			chapter := downloadedChapters[0]
			cbzFilename := packer.GetCBZFilename(title, chapter.Number, chapter.Title)
			if outputDir != "" {
				cbzFilename = filepath.Join(outputDir, filepath.Base(cbzFilename))
				// Create output directory if it doesn't exist
				if err := os.MkdirAll(outputDir, 0755); err != nil {
					return "", fmt.Errorf("failed to create output directory: %w", err)
				}
			}

			// Remove existing file if it exists
			if _, err := os.Stat(cbzFilename); err == nil {
				os.Remove(cbzFilename)
			}

			colors.SavedPrintf("saving to cbz\n")

			packingCallback := func(page, progress int) {
				// Silent packing
			}

			err := packer.ArchiveCBZ(cbzFilename, allFiles, packingCallback)
			if err != nil {
				return "", fmt.Errorf("error creating CBZ file: %w", err)
			}

			output += fmt.Sprintf("Successfully created CBZ file: %s\n", cbzFilename)

			// Convert to other formats if requested
			if convertToAZW3 {
				output += performConversion(cbzFilename, ".azw3")
			}
			if convertToEPUB {
				output += performConversion(cbzFilename, ".epub")
			}
		} else {
			// Multiple chapters - bundle them with chapter-aware naming
			bundleFilename := packer.GetCBZFilename(title, 0, fmt.Sprintf("Chapters %s", chapterRange))
			if outputDir != "" {
				bundleFilename = filepath.Join(outputDir, filepath.Base(bundleFilename))
				// Create output directory if it doesn't exist
				if err := os.MkdirAll(outputDir, 0755); err != nil {
					return "", fmt.Errorf("failed to create output directory: %w", err)
				}
			}

			// Remove existing file if it exists
			if _, err := os.Stat(bundleFilename); err == nil {
				os.Remove(bundleFilename)
			}

			colors.SavedPrintf("saving to cbz\n")

			packingCallback := func(page, progress int) {
				// Silent packing
			}

			err := packer.ArchiveCBZWithChapterInfo(bundleFilename, chapterFiles, packingCallback)
			if err != nil {
				return "", fmt.Errorf("error creating bundled CBZ file: %w", err)
			}

			output += fmt.Sprintf("Successfully created bundled CBZ file: %s\n", bundleFilename)

			// Convert to other formats if requested
			if convertToAZW3 {
				output += performConversion(bundleFilename, ".azw3")
			}
			if convertToEPUB {
				output += performConversion(bundleFilename, ".epub")
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

// performConversion converts a CBZ file to the specified format
func performConversion(cbzFile string, format string) string {
	output := ""

	// Check if ebook-convert is available
	if !converter.IsEbookConvertAvailable() {
		output += colors.Warning("Warning: ebook-convert not found. Please install Calibre to enable format conversion.\n")
		output += "Download from: https://calibre-ebook.com/download\n"
		return output
	}

	conv := converter.NewConverter()
	conv.DeleteSource = false // Keep CBZ file by default

	// Set output directory if specified
	if outputDir := filepath.Dir(cbzFile); outputDir != "." {
		conv.OutputDir = outputDir
	}

	formatName := format[1:] // Remove the dot
	output += fmt.Sprintf("Converting %s to %s format...\n", cbzFile, strings.ToUpper(formatName))

	// Generate output filename
	outputFile := conv.GenerateOutputPath(cbzFile, format)

	// Remove existing output file if it exists
	if _, err := os.Stat(outputFile); err == nil {
		os.Remove(outputFile)
	}

	var result *converter.ConversionResult
	var err error

	if format == ".azw3" {
		result, err = conv.ConvertCBZToAZW3(cbzFile, outputFile)
	} else {
		result, err = conv.ConvertCBZToFormat(cbzFile, outputFile, formatName)
	}

	if err != nil {
		output += colors.Error(fmt.Sprintf("Error converting to %s: %v\n", strings.ToUpper(formatName), err))
		return output
	}

	if result.Success {
		output += fmt.Sprintf("Successfully converted to %s: %s (%d bytes)\n", strings.ToUpper(formatName), result.OutputFile, result.BytesWritten)
	} else {
		output += colors.Error(fmt.Sprintf("%s conversion failed: %v\n", strings.ToUpper(formatName), result.Error))
	}

	return output
}

// expandPath expands ~ to home directory in file paths
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		if usr, err := user.Current(); err == nil {
			return filepath.Join(usr.HomeDir, path[2:])
		}
	}
	return path
}

// listAvailableChapters formats and returns a list of all available chapters
func listAvailableChapters(title string, chapters grabber.Filterables) (string, error) {
	if len(chapters) == 0 {
		return fmt.Sprintf("Title: %s\nNo chapters available.\n", title), nil
	}

	// Collect and sort chapter numbers
	var chapterNumbers []float64
	chapterMap := make(map[float64]grabber.Filterable)
	for _, ch := range chapters {
		num := ch.GetNumber()
		if _, exists := chapterMap[num]; !exists {
			chapterNumbers = append(chapterNumbers, num)
			chapterMap[num] = ch
		}
	}

	// Sort chapters
	for i := 0; i < len(chapterNumbers); i++ {
		for j := i + 1; j < len(chapterNumbers); j++ {
			if chapterNumbers[i] > chapterNumbers[j] {
				chapterNumbers[i], chapterNumbers[j] = chapterNumbers[j], chapterNumbers[i]
			}
		}
	}

	// Build output
	output := fmt.Sprintf("Title: %s\nAvailable chapters (%d total):\n\n", title, len(chapterNumbers))

	for _, num := range chapterNumbers {
		ch := chapterMap[num]
		if num == float64(int64(num)) {
			output += fmt.Sprintf("Chapter %.0f: %s (%s)\n", num, ch.GetTitle(), ch.GetLanguage())
		} else {
			output += fmt.Sprintf("Chapter %.1f: %s (%s)\n", num, ch.GetTitle(), ch.GetLanguage())
		}
	}

	return output, nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: mango <url> [chapter_range] [--azw3] [--epub] [--list] [--output <dir>]")
		fmt.Println("Example: mango https://mangadx.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece")
		fmt.Println("Example: mango https://mangadx.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece --list")
		fmt.Println("Example: mango https://mangadx.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece 1")
		fmt.Println("Example: mango https://mangadx.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece 1-5")
		fmt.Println("Example: mango https://mangadx.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece 1,3,5-10")
		fmt.Println("Example: mango https://mangadx.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece 1-3 --azw3")
		fmt.Println("Example: mango https://mangadx.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece 1-3 --epub")
		fmt.Println("Example: mango https://mangadx.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece 1-3 --azw3 --output ~/Downloads/")
		fmt.Println("")
		fmt.Println("Flags:")
		fmt.Println("  --list           Show all available chapters")
		fmt.Println("  --azw3           Download and convert to AZW3 format for Kindle")
		fmt.Println("  --epub           Download and convert to EPUB format")
		fmt.Println("  --output <dir>   Save files to specified directory (supports ~/)")
		fmt.Println("")
		fmt.Println("Notes:")
		fmt.Println("  • Without format flags, creates CBZ file only")
		fmt.Println("  • Requires Calibre for AZW3/EPUB conversion")
		fmt.Println("  • Files automatically overwrite existing ones")
		fmt.Println("  • Some chapters may be unavailable due to licensing")
		fmt.Println("  • Use --list to see what chapters are actually available")
		return
	}

	url := os.Args[1]
	var chapterRange string
	var outputDir string
	convertToAZW3 := false
	convertToEPUB := false
	listOnly := false

	// Parse remaining arguments
	for i := 2; i < len(os.Args); i++ {
		arg := os.Args[i]
		if arg == "--azw3" || arg == "--awz3" {
			convertToAZW3 = true
		} else if arg == "--epub" {
			convertToEPUB = true
		} else if arg == "--list" {
			listOnly = true
		} else if arg == "--output" && i+1 < len(os.Args) {
			outputDir = expandPath(os.Args[i+1])
			i++ // Skip the next argument since it's the output directory
		} else if chapterRange == "" && !strings.HasPrefix(arg, "--") {
			chapterRange = arg
		}
	}

	// Auto-enable download and CBZ if conversion format is specified
	download := convertToAZW3 || convertToEPUB
	saveCBZ := convertToAZW3 || convertToEPUB

	// If no conversion format specified, just download and create CBZ
	if !convertToAZW3 && !convertToEPUB {
		download = true
		saveCBZ = true
	}

	content, err := FetchURLContent(url, chapterRange, download, saveCBZ, convertToAZW3, convertToEPUB, outputDir, listOnly)
	if err != nil {
		colors.ErrorPrintf("Error: %v\n", err)
		return
	}

	fmt.Println(content)
}
