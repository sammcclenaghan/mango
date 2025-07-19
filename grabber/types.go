package grabber

// Settings holds configuration for the grabber
type Settings struct {
	Language string
	Bundle   bool
}

// Page represents a single manga page
type Page struct {
	Number int64
	URL    string
}

// Chapter represents a manga chapter
type Chapter struct {
	Number     float64
	Title      string
	Language   string
	PagesCount int64
	Pages      []Page
}

// Filterable interface for objects that can be filtered by number
type Filterable interface {
	GetNumber() float64
	GetLanguage() string
	GetTitle() string
}

// Filterables is a slice of Filterable objects
type Filterables []Filterable

// GetNumber implements Filterable for Chapter
func (c Chapter) GetNumber() float64 {
	return c.Number
}

// GetLanguage implements Filterable for Chapter
func (c Chapter) GetLanguage() string {
	return c.Language
}

// GetTitle implements Filterable for Chapter
func (c Chapter) GetTitle() string {
	return c.Title
}

// Grabber is the base grabber struct
type Grabber struct {
	URL      string
	Settings Settings
}

// BaseUrl returns the base URL of the site
func (g *Grabber) BaseUrl() string {
	// Extract base URL from full URL
	// This is a simple implementation, could be enhanced
	return g.URL
}

// GrabberInterface defines the interface that all grabbers must implement
type GrabberInterface interface {
	Test() (bool, error)
	FetchTitle() (string, error)
	FetchChapters() (Filterables, []error)
	FetchChapter(Filterable) (*Chapter, error)
}
