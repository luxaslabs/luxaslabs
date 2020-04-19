package scraper

import (
	"fmt"
	"sync"

	"github.com/gocolly/colly"
	log "github.com/sirupsen/logrus"
)

type HookFn func(e *colly.HTMLElement, data interface{}) (*string, error)

type Hook struct {
	DOMPath string
	Handler HookFn
}

type Scraper interface {
	// From BaseScraper:
	Name() string

	Extensions() map[string]Extension
	AddExtension(...Extension) error

	Children() map[string]Scraper
	AddChildren(...Scraper) error

	// Per-Scraper:
	Hooks() []Hook
	InitialData() interface{}
}

type Extension interface {
	Name() string
	Hook() Hook
}

func NewBaseScraper(name string, children []Scraper, exts []Extension) (*BaseScraper, error) {
	bs := &BaseScraper{
		name:     name,
		children: make(map[string]Scraper),
		exts:     make(map[string]Extension),
	}
	for _, child := range children {
		if err := bs.AddChildren(child); err != nil {
			return nil, err
		}
	}
	for _, ext := range exts {
		if err := bs.AddExtension(ext); err != nil {
			return nil, err
		}
	}
	return bs, nil
}

type BaseScraper struct {
	name     string
	children map[string]Scraper
	exts     map[string]Extension
}

func (bs *BaseScraper) Name() string {
	return bs.name
}

func (bs *BaseScraper) Children() map[string]Scraper {
	return bs.children
}

func (bs *BaseScraper) AddChildren(scrapers ...Scraper) error {
	for _, s := range scrapers {
		if _, ok := bs.children[s.Name()]; ok {
			return fmt.Errorf("children with name %s already exists!", s.Name())
		}
		bs.children[s.Name()] = s
	}
	return nil
}

func (bs *BaseScraper) Extensions() map[string]Extension {
	return bs.exts
}

func (bs *BaseScraper) AddExtension(exts ...Extension) (returnerr error) {
	for _, e := range exts {
		if _, ok := bs.exts[e.Name()]; ok {
			returnerr = fmt.Errorf("extension with name %s already exists!", e.Name())
			log.Error(returnerr)
		}
		bs.exts[e.Name()] = e
	}
	for _, c := range bs.children {
		if err := c.AddExtension(exts...); err != nil {
			returnerr = fmt.Errorf("child scraper %s already has extension: %v", c.Name(), err)
			log.Error(returnerr)
		}
	}
	return
}

func Scrape(url string, s Scraper) (interface{}, error) {
	c := colly.NewCollector()
	mux := &sync.Mutex{}
	data := s.InitialData()

	allHooks := s.Hooks()
	for _, ext := range s.Extensions() {
		allHooks = append(allHooks, ext.Hook())
	}

	errs := []error{}
	for _, h := range allHooks {
		func(hook Hook) {
			c.OnHTML(hook.DOMPath, func(e *colly.HTMLElement) {
				mux.Lock()

				log.Debugf("DOMPath: %q, URL: %q", hook.DOMPath, e.Request.URL)
				next, err := hook.Handler(e, data)
				if err != nil {
					log.Errorf("error while handling dompath %q for request %q: %v", hook.DOMPath, e.Request.URL, err)
					errs = append(errs, err)
				}
				mux.Unlock()

				if next != nil {
					c.Visit(*next)
				}
			})
		}(h)
	}
	c.OnRequest(func(r *colly.Request) {
		log.Infof("%s visiting page %q", s.Name(), r.URL)
	})

	if err := c.Visit(url); err != nil {
		return nil, err
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("errors occured during scraping: %v", errs)
	}

	return data, nil
}
