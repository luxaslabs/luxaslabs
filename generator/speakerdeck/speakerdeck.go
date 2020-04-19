package speakerdeck

import (
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gocolly/colly"
	"github.com/luxaslabs/luxaslabs/generator/scraper"
	"github.com/luxaslabs/luxaslabs/generator/speakerdeck/types"
	log "github.com/sirupsen/logrus"
)

var linkRegexp = regexp.MustCompile(`http[s]?://[a-zA-Z-_/0-9\.#=&]*`)

var _ scraper.Scraper = &UserScraper{}

func NewUserScraper() *UserScraper {
	bs, _ := scraper.NewBaseScraper("UserScraper", []scraper.Scraper{NewTalkScraper()}, nil)
	return &UserScraper{bs}
}

type UserScraper struct {
	*scraper.BaseScraper
}

func (s *UserScraper) ScrapeUser(userID string) (*types.User, error) {
	fullURL := fmt.Sprintf("https://speakerdeck.com/%s", userID)
	data, err := scraper.Scrape(fullURL, s)
	if err != nil {
		return nil, err
	}
	user := data.(*types.User) // error handling
	return user, nil
}

func (s *UserScraper) Hooks() []scraper.Hook {
	return []scraper.Hook{
		{
			DOMPath: ".sd-main h1.m-0",
			Handler: s.onName,
		},
		{
			DOMPath: ".container a[href][title]",
			Handler: s.onTalkLink,
		},
		{
			DOMPath: ".next .page-link[rel='next']",
			Handler: s.onNextPage,
		},
	}
}

func (s *UserScraper) InitialData() interface{} {
	return types.NewUser()
}

func (s *UserScraper) onName(e *colly.HTMLElement, data interface{}) (*string, error) {
	u := data.(*types.User)
	u.Name = e.Text
	return nil, nil
}

func (s *UserScraper) onTalkLink(e *colly.HTMLElement, data interface{}) (*string, error) {
	u := data.(*types.User)

	href := e.Attr("href") // i.e. "/userFoo/talkFoo"
	parts := strings.Split(href, "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid talk link: %q", href)
	}

	ts := s.Children()["TalkScraper"].(*TalkScraper)
	talk, err := ts.ScrapeTalk(parts[1], parts[2])
	if err != nil {
		return nil, err
	}

	u.Talks = append(u.Talks, *talk)
	// Always ensure proper ordering
	sort.Sort(u.Talks)

	return nil, nil
}

func (s *UserScraper) onNextPage(e *colly.HTMLElement, _ interface{}) (*string, error) {
	href := e.Attr("href")
	if len(href) > 0 {
		nextURL := fmt.Sprintf("https://speakerdeck.com%s", e.Attr("href"))
		return &nextURL, nil
	}
	return nil, nil
}

func NewTalkScraper() *TalkScraper {
	bs, _ := scraper.NewBaseScraper("TalkScraper", nil, nil)
	return &TalkScraper{bs}
}

type TalkScraper struct {
	*scraper.BaseScraper
}

func (s *TalkScraper) ScrapeTalk(userID, talkID string) (*types.Talk, error) {
	fullURL := fmt.Sprintf("https://speakerdeck.com/%s/%s", userID, talkID)
	parsedURL, err := url.Parse(fullURL)
	if err != nil {
		return nil, err
	}

	data, err := scraper.Scrape(fullURL, s)
	if err != nil {
		return nil, err
	}

	talk := data.(*types.Talk) // error handling
	talk.Link = *parsedURL
	return talk, nil
}

func (s *TalkScraper) Hooks() []scraper.Hook {
	return []scraper.Hook{
		{
			DOMPath: ".container h1.mb-4",
			Handler: s.onTitle,
		},
		{
			DOMPath: ".col-auto.text-muted",
			Handler: s.onDate,
		},
		{
			DOMPath: ".deck-description.mb-4 p",
			Handler: s.onDescription,
		},
		{
			DOMPath: ".speakerdeck-embed",
			Handler: s.onDataID,
		},
	}
}

func (s *TalkScraper) InitialData() interface{} {
	return types.NewTalk()
}

func (s *TalkScraper) onTitle(e *colly.HTMLElement, data interface{}) (*string, error) {
	t := data.(*types.Talk)
	t.Title = e.Text
	return nil, nil
}

func (s *TalkScraper) onDataID(e *colly.HTMLElement, data interface{}) (*string, error) {
	t := data.(*types.Talk)
	t.DataID = e.Attr("data-id")
	return nil, nil
}

func (s *TalkScraper) onDate(e *colly.HTMLElement, data interface{}) (*string, error) {
	t := data.(*types.Talk)
	// this element contains the date of the talk
	date := e.Text
	// sanitize the text
	date = strings.Trim(strings.ReplaceAll(strings.ReplaceAll(date, ",", ""), "\n", ""), " ")
	// and parse it
	d, err := time.Parse("January 02 2006", date)
	if err != nil {
		return nil, err
	}
	t.Date = d
	return nil, nil
}

func (s *TalkScraper) onDescription(e *colly.HTMLElement, data interface{}) (*string, error) {
	t := data.(*types.Talk)
	links := linkRegexp.FindStringSubmatch(e.Text)
	for _, link := range links {
		parsedLink, err := url.Parse(link)
		if err != nil {
			log.Warnf("Could not parse link %q", link)
			continue
		}
		t.ExtraLinks[parsedLink.Host] = *parsedLink
	}

	if strings.Contains(e.Text, "Hide: true") {
		t.Hide = true
	}

	return nil, nil
}
