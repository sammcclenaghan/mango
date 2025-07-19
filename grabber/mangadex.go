package grabber

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"time"

	"github.sammcclenaghan.com/mango/http"
)

// Mangadx is a grabber for mangadex.org
type Mangadx struct {
	*Grabber
	title string
	// rateLimiter rate limiter for the FetchChapter method. This call uses the '/at-home' endpoint which has a rate limit
	// of 40 calls per minute, if we exceed this limit we get a 429, and the consequent chapters fail. This may eventually
	// lead to an IP ban.
	rateLimiter <-chan time.Time
}

func NewMangadx(g *Grabber) *Mangadx {
	// we set the rate limit at 39 calls per minute instead of 40 to make sure the rate limit is under the threshold,
	// otherwise we occasionally get hit by the rate limiter.
	return &Mangadx{Grabber: g, rateLimiter: time.Tick(time.Minute / 39)}
}

// MangadxChapter represents a MangaDx Chapter
type MangadxChapter struct {
	Chapter
	Id string
}

// Test checks if the site is MangaDx
func (m *Mangadx) Test() (bool, error) {
	re := regexp.MustCompile(`mangadex\.org`)
	return re.MatchString(m.URL), nil
}

// GetTitle returns the title of the manga
func (m *Mangadx) FetchTitle() (string, error) {
	if m.title != "" {
		return m.title, nil
	}

	id := getUuid(m.URL)

	rbody, err := http.Get(http.RequestParams{
		URL:     "https://api.mangadex.org/manga/" + id,
		Referer: m.BaseUrl(),
	})
	if err != nil {
		return "", err
	}
	defer rbody.Close()

	// decode json response
	body := mangadxManga{}
	if err = json.NewDecoder(rbody).Decode(&body); err != nil {
		return "", err
	}

	// fetch the title in the requested language
	if m.Settings.Language != "" {
		trans := body.Data.Attributes.AltTitles.GetTitleByLang(m.Settings.Language)

		if trans != "" {
			m.title = trans
			return m.title, nil
		}
	}

	// fallback to english
	m.title = body.Data.Attributes.Title["en"]

	return m.title, nil
}

// FetchChapters returns the chapters of the manga
func (m Mangadx) FetchChapters() (chapters Filterables, errs []error) {
	id := getUuid(m.URL)

	baseOffset := 500
	var fetchChaps func(int)

	fetchChaps = func(offset int) {
		uri := fmt.Sprintf("https://api.mangadex.org/manga/%s/feed", id)
		params := url.Values{}
		params.Add("limit", fmt.Sprint(baseOffset))
		params.Add("order[volume]", "asc")
		params.Add("order[chapter]", "asc")
		params.Add("offset", fmt.Sprint(offset))
		if m.Settings.Language != "" {
			params.Add("translatedLanguage[]", m.Settings.Language)
		}
		uri = fmt.Sprintf("%s?%s", uri, params.Encode())

		rbody, err := http.Get(http.RequestParams{URL: uri})
		if err != nil {
			errs = append(errs, err)
			return
		}
		defer rbody.Close()
		// parse json body
		body := mangadxFeed{}
		if err = json.NewDecoder(rbody).Decode(&body); err != nil {
			errs = append(errs, err)
			return
		}

		for _, c := range body.Data {
			num, _ := strconv.ParseFloat(c.Attributes.Chapter, 64)
			chapters = append(chapters, &MangadxChapter{
				Chapter{
					Number:     num,
					Title:      c.Attributes.Title,
					Language:   c.Attributes.TranslatedLanguage,
					PagesCount: c.Attributes.Pages,
				},
				c.Id,
			})
		}

		if len(body.Data) > 0 {
			fetchChaps(offset + baseOffset)
		}
	}
	// initial call
	fetchChaps(0)

	return
}

// FetchChapter fetches a chapter and its pages
func (m Mangadx) FetchChapter(f Filterable) (*Chapter, error) {
	<-m.rateLimiter
	chap := f.(*MangadxChapter)
	// download json
	rbody, err := http.Get(http.RequestParams{
		URL: "https://api.mangadex.org/at-home/server/" + chap.Id,
	})
	if err != nil {
		return nil, err
	}
	defer rbody.Close()
	// parse json body
	body := mangadxPagesFeed{}
	if err = json.NewDecoder(rbody).Decode(&body); err != nil {
		return nil, err
	}

	pcount := len(body.Chapter.Data)

	chapter := &Chapter{
		Title:      fmt.Sprintf("Chapter %04d %s", int64(f.GetNumber()), chap.Title),
		Number:     f.GetNumber(),
		PagesCount: int64(pcount),
		Language:   chap.Language,
	}

	// create pages
	for i, p := range body.Chapter.Data {
		num := i + 1
		chapter.Pages = append(chapter.Pages, Page{
			Number: int64(num),
			URL:    body.BaseUrl + path.Join("/data", body.Chapter.Hash, p),
		})
	}

	return chapter, nil
}

// getUuid extracts the UUID from a MangaDx URL
func getUuid(urlStr string) string {
	re := regexp.MustCompile(`[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`)
	return re.FindString(urlStr)
}

// mangadxManga represents the Manga json object
type mangadxManga struct {
	Id   string
	Data struct {
		Attributes struct {
			Title     map[string]string
			AltTitles altTitles
		}
	}
}

// altTitles is a slice of maps with the language as key and the title as value
type altTitles []map[string]string

// GetTitleByLang returns the title in the given language (or empty if string is not found)
func (a altTitles) GetTitleByLang(lang string) string {
	for _, t := range a {
		val, ok := t[lang]
		if ok {
			return val
		}
	}
	return ""
}

// mangadxFeed represents the json object returned by the feed endpoint
type mangadxFeed struct {
	Data []struct {
		Id         string
		Attributes struct {
			Volume             string
			Chapter            string
			Title              string
			TranslatedLanguage string
			Pages              int64
		}
	}
}

// mangadxPagesFeed represents the json object returned by the pages endpoint
type mangadxPagesFeed struct {
	BaseUrl string
	Chapter struct {
		Hash      string
		Data      []string
		DataSaver []string
	}
}
