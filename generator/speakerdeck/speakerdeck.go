package speakerdeck

import (
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly"
	log "github.com/sirupsen/logrus"
)

var linkRegexp = regexp.MustCompile(`http[s]?://[a-zA-Z-_/0-9\.#=&]*`)

type BaseScraper struct {
	*colly.Collector
	dataMux      *sync.Mutex
	currentError error
}

func NewBaseScraper(name string) *BaseScraper {
	bs := &BaseScraper{colly.NewCollector(), &sync.Mutex{}, nil}
	bs.OnRequest(func(r *colly.Request) {
		log.Infof("%s visiting page %q", name, r.URL)
	})

	return bs
}

func (s *BaseScraper) RegisterHook(domPath string, hookFn func(*colly.HTMLElement) error) {
	s.OnHTML(domPath, func(e *colly.HTMLElement) {
		if err := hookFn(e); err != nil {
			s.currentError = err
			return
		}
	})
}

func (bs *BaseScraper) StartScraping(fullURL string) error {
	// Initialize the data for the scraper and set the error to nil
	bs.currentError = nil

	// Start scraping
	if err := bs.Visit(fullURL); err != nil {
		return err
	}

	// If an error has occured in some hook, return that, otherwise nil
	return bs.currentError
}

type UserPageScraper struct {
	*BaseScraper
	talkPageScraper *TalkPageScraper
	currentUser     *User
}

func NewUserPageScraper() *UserPageScraper {
	userPageScraper := &UserPageScraper{NewBaseScraper("UserPageScraper"), NewTalkPageScraper(), nil}
	userPageScraper.RegisterHook(".sd-main h1.m-0", userPageScraper.onName)
	userPageScraper.RegisterHook(".container a[href][title]", userPageScraper.onTalkLink)
	//	userPageScraper.RegisterHook(".container a > div[data-id]", userPageScraper.onDataID)
	userPageScraper.RegisterHook(".next .page-link[rel='next']", userPageScraper.onNextPage)
	return userPageScraper
}

func (s *UserPageScraper) onName(e *colly.HTMLElement) error {
	s.currentUser.Name = e.Text
	return nil
}

func (s *UserPageScraper) onTalkLink(e *colly.HTMLElement) error {
	href := e.Attr("href") // i.e. "/userFoo/talkFoo"
	parts := strings.Split(href, "/")
	if len(parts) != 3 {
		return fmt.Errorf("invalid talk link: %q", href)
	}
	talk, err := s.talkPageScraper.Scrape(parts[1], parts[2])
	if err != nil {
		return err
	}

	// Get the data-id attr used for embedding
	talk.DataID = e.ChildAttr("div[data-id]", "data-id")

	s.currentUser.Talks = append(s.currentUser.Talks, *talk)
	return nil
}

func (s *UserPageScraper) onNextPage(e *colly.HTMLElement) error {
	nextURL := fmt.Sprintf("https://speakerdeck.com%s", e.Attr("href"))
	s.Visit(nextURL)
	return nil
}

func (s *UserPageScraper) Scrape(userID string) (*User, error) {
	fullURL := fmt.Sprintf("https://speakerdeck.com/%s", userID)

	// wait for our turn
	s.dataMux.Lock()
	defer s.dataMux.Unlock()

	// Initialize an empty user for the hooks to write to
	s.currentUser = &User{}

	if err := s.StartScraping(fullURL); err != nil {
		return nil, err
	}

	// Sort the talks before returning them
	sort.Sort(s.currentUser.Talks)

	return s.currentUser, nil
}

type TalkPageScraper struct {
	*BaseScraper
	currentTalk *Talk
}

func NewTalkPageScraper() *TalkPageScraper {
	talkPageScraper := &TalkPageScraper{NewBaseScraper("TalkPageScraper"), nil}
	talkPageScraper.RegisterHook(".col-auto.text-muted", talkPageScraper.onDate)
	talkPageScraper.RegisterHook(".deck-description.mb-4 p", talkPageScraper.onLinks)
	talkPageScraper.RegisterHook(".container h1.mb-4", talkPageScraper.onTitle)
	return talkPageScraper
}

func (s *TalkPageScraper) Scrape(userID, talkID string) (*Talk, error) {
	fullURL := fmt.Sprintf("https://speakerdeck.com/%s/%s", userID, talkID)

	// wait for our turn
	s.dataMux.Lock()
	defer s.dataMux.Unlock()
	// Initialize an empty talk for the hooks to write to
	talkURL, err := url.Parse(fullURL)
	if err != nil {
		return nil, err
	}
	s.currentTalk = NewTalk()
	s.currentTalk.Link = *talkURL

	if err := s.StartScraping(fullURL); err != nil {
		return nil, err
	}

	return s.currentTalk, nil
}

func (s *TalkPageScraper) onTitle(e *colly.HTMLElement) error {
	s.currentTalk.Title = e.Text
	return nil
}

func (s *TalkPageScraper) onDate(e *colly.HTMLElement) error {
	// this element contains the date of the talk
	date := e.Text
	// sanitize the text
	date = strings.Trim(strings.ReplaceAll(strings.ReplaceAll(date, ",", ""), "\n", ""), " ")
	// and parse it
	d, err := time.Parse("January 02 2006", date)
	if err != nil {
		return err
	}
	s.currentTalk.Date = d
	return nil
}

func (s *TalkPageScraper) onLinks(e *colly.HTMLElement) error {
	links := linkRegexp.FindStringSubmatch(e.Text)
	for _, link := range links {
		parsedLink, err := url.Parse(link)
		if err != nil {
			log.Warnf("Could not parse link %q", link)
			continue
		}
		s.currentTalk.ExtraLinks[parsedLink.Host] = *parsedLink
	}
	return nil
}
