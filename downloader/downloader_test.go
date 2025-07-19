package downloader

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.sammcclenaghan.com/mango/grabber"
	httpPkg "github.sammcclenaghan.com/mango/http"
)

// MockGrabber implements GrabberInterface for testing
type MockGrabber struct {
	url string
}

func (m *MockGrabber) Test() (bool, error) {
	return true, nil
}

func (m *MockGrabber) FetchTitle() (string, error) {
	return "Test Manga", nil
}

func (m *MockGrabber) FetchChapters() (grabber.Filterables, []error) {
	return nil, nil
}

func (m *MockGrabber) FetchChapter(f grabber.Filterable) (*grabber.Chapter, error) {
	return nil, nil
}

func TestFetchFile_Success(t *testing.T) {
	// Create test server that returns image data
	testData := []byte("fake image data")
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write(testData)
	}))
	defer ts.Close()

	// Test fetching a file
	file, err := FetchFile(httpPkg.RequestParams{
		URL: ts.URL,
	}, 1)

	if err != nil {
		t.Fatalf("FetchFile() error = %v", err)
	}

	if file == nil {
		t.Fatal("FetchFile() returned nil file")
	}

	if file.Page != 1 {
		t.Errorf("FetchFile() page = %d, want %d", file.Page, 1)
	}

	if !bytes.Equal(file.Data, testData) {
		t.Errorf("FetchFile() data mismatch")
	}
}

func TestFetchFile_HTTPError(t *testing.T) {
	// Create test server that returns 404
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	// Test fetching a file that returns 404
	file, err := FetchFile(httpPkg.RequestParams{
		URL: ts.URL,
	}, 1)

	if err == nil {
		t.Error("FetchFile() expected error for 404, but got none")
	}

	if file != nil {
		t.Error("FetchFile() expected nil file for error case")
	}
}

func TestFetchFile_InvalidURL(t *testing.T) {
	// Test with invalid URL
	file, err := FetchFile(httpPkg.RequestParams{
		URL: "invalid-url",
	}, 1)

	if err == nil {
		t.Error("FetchFile() expected error for invalid URL, but got none")
	}

	if file != nil {
		t.Error("FetchFile() expected nil file for error case")
	}
}

func TestFetchChapter_Success(t *testing.T) {
	// Create test server that returns different data for each page
	pageData := map[string][]byte{
		"/page1.jpg": []byte("page 1 data"),
		"/page2.jpg": []byte("page 2 data"),
		"/page3.jpg": []byte("page 3 data"),
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if data, exists := pageData[r.URL.Path]; exists {
			w.Header().Set("Content-Type", "image/jpeg")
			w.Write(data)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	// Create test chapter
	chapter := &grabber.Chapter{
		Number:     1,
		Title:      "Test Chapter",
		PagesCount: 3,
		Pages: []grabber.Page{
			{Number: 1, URL: ts.URL + "/page1.jpg"},
			{Number: 2, URL: ts.URL + "/page2.jpg"},
			{Number: 3, URL: ts.URL + "/page3.jpg"},
		},
	}

	// Track progress
	var progressCalls []int
	progressCallback := func(page, progress int, err error) {
		if err != nil {
			t.Errorf("Unexpected error in progress callback: %v", err)
		}
		progressCalls = append(progressCalls, progress)
	}

	// Create mock grabber
	mockGrabber := &MockGrabber{url: ts.URL}

	// Test fetching chapter
	files, err := FetchChapter(mockGrabber, chapter, progressCallback)
	if err != nil {
		t.Fatalf("FetchChapter() error = %v", err)
	}

	// Verify results
	if len(files) != 3 {
		t.Errorf("FetchChapter() returned %d files, want 3", len(files))
	}

	// Verify files are sorted by page number
	for i, file := range files {
		expectedPage := uint(i + 1)
		if file.Page != expectedPage {
			t.Errorf("File %d has page %d, want %d", i, file.Page, expectedPage)
		}

		expectedData := pageData[chapter.Pages[i].URL[len(ts.URL):]]
		if !bytes.Equal(file.Data, expectedData) {
			t.Errorf("File %d data mismatch", i)
		}
	}

	// Verify progress was called
	if len(progressCalls) != 3 {
		t.Errorf("Expected 3 progress calls, got %d", len(progressCalls))
	}
}

func TestFetchChapter_WithError(t *testing.T) {
	// Create test server that returns 404 for one page
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/page2.jpg" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write([]byte("test data"))
	}))
	defer ts.Close()

	// Create test chapter with one failing page
	chapter := &grabber.Chapter{
		Number:     1,
		Title:      "Test Chapter",
		PagesCount: 2,
		Pages: []grabber.Page{
			{Number: 1, URL: ts.URL + "/page1.jpg"},
			{Number: 2, URL: ts.URL + "/page2.jpg"}, // This will fail
		},
	}

	// Track errors
	var errorCount int
	progressCallback := func(page, progress int, err error) {
		if err != nil {
			errorCount++
		}
	}

	// Create mock grabber
	mockGrabber := &MockGrabber{url: ts.URL}

	// Test fetching chapter
	files, err := FetchChapter(mockGrabber, chapter, progressCallback)

	// Should return error when a page fails
	if err == nil {
		t.Error("FetchChapter() expected error when page fails, but got none")
	}

	if files != nil {
		t.Error("FetchChapter() expected nil files when error occurs")
	}

	if errorCount == 0 {
		t.Error("Expected at least one error callback, got none")
	}
}

func TestFetchChapter_EmptyChapter(t *testing.T) {
	// Create empty chapter
	chapter := &grabber.Chapter{
		Number:     1,
		Title:      "Empty Chapter",
		PagesCount: 0,
		Pages:      []grabber.Page{},
	}

	progressCallback := func(page, progress int, err error) {
		t.Error("Progress callback should not be called for empty chapter")
	}

	// Create mock grabber
	mockGrabber := &MockGrabber{url: "http://example.com"}

	// Test fetching empty chapter
	files, err := FetchChapter(mockGrabber, chapter, progressCallback)
	if err != nil {
		t.Errorf("FetchChapter() error = %v", err)
	}

	if len(files) != 0 {
		t.Errorf("FetchChapter() returned %d files for empty chapter, want 0", len(files))
	}
}

func TestFetchChapter_Concurrency(t *testing.T) {
	// Create test server with artificial delay
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond) // Simulate slow response
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write([]byte("test data"))
	}))
	defer ts.Close()

	// Create chapter with multiple pages
	numPages := 10
	pages := make([]grabber.Page, numPages)
	for i := 0; i < numPages; i++ {
		pages[i] = grabber.Page{
			Number: int64(i + 1),
			URL:    ts.URL + "/page" + fmt.Sprintf("%d", i+1) + ".jpg",
		}
	}

	chapter := &grabber.Chapter{
		Number:     1,
		Title:      "Concurrent Test Chapter",
		PagesCount: int64(numPages),
		Pages:      pages,
	}

	progressCallback := func(page, progress int, err error) {
		if err != nil {
			t.Errorf("Unexpected error in progress callback: %v", err)
		}
	}

	// Create mock grabber
	mockGrabber := &MockGrabber{url: ts.URL}

	// Measure time to ensure concurrency is working
	start := time.Now()
	files, err := FetchChapter(mockGrabber, chapter, progressCallback)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("FetchChapter() error = %v", err)
	}

	if len(files) != numPages {
		t.Errorf("FetchChapter() returned %d files, want %d", len(files), numPages)
	}

	// With concurrency, it should take less time than sequential (10 * 100ms = 1s)
	maxExpectedDuration := time.Duration(numPages) * 100 * time.Millisecond
	if duration >= maxExpectedDuration {
		t.Errorf("FetchChapter() took %v, expected less than %v (concurrency not working)", duration, maxExpectedDuration)
	}
}
