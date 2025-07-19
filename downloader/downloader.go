package downloader

import (
	"fmt"
	"io"
	"sort"
	"sync"

	"github.sammcclenaghan.com/mango/grabber"
	"github.sammcclenaghan.com/mango/http"
)

// File represents a downloaded file
type File struct {
	Data []byte
	Page uint
}

// ProgressCallback is a function type for progress updates with optional error
type ProgressCallback func(page, progress int, err error)

// FetchChapter downloads all the pages of a chapter
func FetchChapter(site grabber.GrabberInterface, chapter *grabber.Chapter, onprogress ProgressCallback) (files []*File, err error) {
	if len(chapter.Pages) == 0 {
		return []*File{}, nil
	}

	wg := sync.WaitGroup{}
	guard := make(chan struct{}, 5) // Default max concurrency of 5
	errChan := make(chan error, 1)
	done := make(chan bool)
	fileChan := make(chan *File, len(chapter.Pages))
	var downloadErr error

	for i, page := range chapter.Pages {
		guard <- struct{}{}
		wg.Add(1)
		go func(page grabber.Page, idx int) {
			defer wg.Done()

			file, err := FetchFile(http.RequestParams{
				URL: page.URL,
			}, uint(page.Number))

			if err != nil {
				select {
				case errChan <- fmt.Errorf("page %d: %w", page.Number, err):
					onprogress(idx, idx, err)
				default:
				}
				<-guard
				return
			}

			fileChan <- file
			onprogress(1, idx, nil) // Progress by 1 page at a time
			<-guard
		}(page, i)
	}

	go func() {
		wg.Wait()
		close(done)
		close(fileChan)
	}()

	// Collect files from channel
	files = make([]*File, 0, len(chapter.Pages))

	collecting := true
	for collecting {
		select {
		case err := <-errChan:
			downloadErr = err
			collecting = false
		case file := <-fileChan:
			if file != nil {
				files = append(files, file)
			}
		case <-done:
			collecting = false
		}
	}

	// Collect any remaining files from the channel
	for file := range fileChan {
		if file != nil {
			files = append(files, file)
		}
	}

	if downloadErr != nil {
		return nil, downloadErr
	}

	// sort files by page number
	sort.SliceStable(files, func(i, j int) bool {
		return files[i].Page < files[j].Page
	})

	return
}

// FetchFile gets an online file returning a new *File with its contents
func FetchFile(params http.RequestParams, page uint) (file *File, err error) {
	body, err := http.Get(params)
	if err != nil {
		// TODO: should retry at least once (configurable)
		return
	}

	defer body.Close()

	data, err := io.ReadAll(body)
	if err != nil {
		return
	}

	file = &File{
		Data: data,
		Page: page,
	}

	return
}
