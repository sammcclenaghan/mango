package main

import (
	"fmt"
	"os"
	"strconv"

	"github.sammcclenaghan.com/mango/downloader"
	"github.sammcclenaghan.com/mango/grabber"
	"github.sammcclenaghan.com/mango/packer"
)

// FetchURLContent fetches the content from the given URL and returns it as a string.
func FetchURLContent(url string, chapterNum string, download bool, saveCBZ bool) (string, error) {
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

	// If a specific chapter is requested, fetch its pages
	if chapterNum != "" {
		fmt.Printf("Debug: Looking for chapter %s\n", chapterNum)
		fmt.Printf("Debug: Available chapters: %d\n", len(chapters))
		return fetchSpecificChapter(mangadx, chapters, chapterNum, title, download, saveCBZ)
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

// fetchSpecificChapter fetches pages for a specific chapter
func fetchSpecificChapter(mangadx *grabber.Mangadx, chapters grabber.Filterables, chapterNum string, title string, download bool, saveCBZ bool) (string, error) {
	// Parse the requested chapter number
	targetChapter, err := strconv.ParseFloat(chapterNum, 64)
	if err != nil {
		return "", fmt.Errorf("invalid chapter number: %s", chapterNum)
	}

	// Find the matching chapter
	var selectedChapter grabber.Filterable
	for _, chapter := range chapters {
		if chapter.GetNumber() == targetChapter {
			selectedChapter = chapter
			break
		}
	}

	if selectedChapter == nil {
		// List available chapters for debugging
		availableChapters := ""
		for _, ch := range chapters {
			availableChapters += fmt.Sprintf("%.1f ", ch.GetNumber())
		}
		return "", fmt.Errorf("chapter %.1f not found. Available chapters: %s", targetChapter, availableChapters)
	}

	// Debug: Print chapter ID before fetching
	if mangadxChap, ok := selectedChapter.(*grabber.MangadxChapter); ok {
		fmt.Printf("Debug: Fetching chapter ID: %s\n", mangadxChap.Id)
	}

	// Fetch the chapter with its pages
	chapterWithPages, err := mangadx.FetchChapter(selectedChapter)
	if err != nil {
		return "", fmt.Errorf("error fetching chapter pages: %w", err)
	}

	// Build detailed output
	output := fmt.Sprintf("Title: %s\n", title)
	output += fmt.Sprintf("Chapter: %.1f - %s\n", chapterWithPages.Number, chapterWithPages.Title)
	output += fmt.Sprintf("Language: %s\n", chapterWithPages.Language)
	output += fmt.Sprintf("Pages: %d\n\n", chapterWithPages.PagesCount)

	if download {
		// Download the chapter pages
		output += "Downloading pages...\n"
		progressCallback := func(page, progress int, err error) {
			if err != nil {
				fmt.Printf("Error downloading page %d: %v\n", page, err)
			} else {
				fmt.Printf("Downloaded page %d\n", progress+1)
			}
		}

		files, err := downloader.FetchChapter(mangadx, chapterWithPages, progressCallback)
		if err != nil {
			return "", fmt.Errorf("error downloading chapter: %w", err)
		}

		output += fmt.Sprintf("\nSuccessfully downloaded %d pages\n", len(files))

		// Save to CBZ if requested
		if saveCBZ {
			cbzFilename := packer.GetCBZFilename(title, chapterWithPages.Number, chapterWithPages.Title)
			output += fmt.Sprintf("Creating CBZ file: %s\n", cbzFilename)

			packingCallback := func(page, progress int) {
				fmt.Printf("Packing page %d into CBZ...\n", progress+1)
			}

			err := packer.ArchiveCBZ(cbzFilename, files, packingCallback)
			if err != nil {
				return "", fmt.Errorf("error creating CBZ file: %w", err)
			}

			output += fmt.Sprintf("Successfully created CBZ file: %s\n", cbzFilename)
		} else {
			// List downloaded file sizes
			for _, file := range files {
				output += fmt.Sprintf("Page %d: %d bytes\n", file.Page, len(file.Data))
			}
		}
	} else {
		// Just list page URLs
		output += "Page URLs:\n"
		for _, page := range chapterWithPages.Pages {
			output += fmt.Sprintf("Page %d: %s\n", page.Number, page.URL)
		}
	}

	return output, nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: mango <url> [chapter_number] [--download] [--cbz]")
		fmt.Println("Example: mango https://mangadex.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece")
		fmt.Println("Example: mango https://mangadex.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece 1")
		fmt.Println("Example: mango https://mangadex.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece 1 --download")
		fmt.Println("Example: mango https://mangadex.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece 1 --download --cbz")
		fmt.Println("Use --download flag to actually download pages (default: just list URLs)")
		fmt.Println("Use --cbz flag to save downloaded pages as CBZ file (requires --download)")
		return
	}

	url := os.Args[1]
	var chapterNum string
	download := false
	saveCBZ := false

	// Parse remaining arguments
	for i := 2; i < len(os.Args); i++ {
		arg := os.Args[i]
		if arg == "--download" {
			download = true
		} else if arg == "--cbz" {
			saveCBZ = true
		} else if chapterNum == "" {
			chapterNum = arg
		}
	}

	// Validate flags
	if saveCBZ && !download {
		fmt.Println("Error: --cbz flag requires --download flag")
		return
	}

	content, err := FetchURLContent(url, chapterNum, download, saveCBZ)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println(content)
}
