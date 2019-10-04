package main

import (
	"fmt"
	"os"
	"time"
	"strings"
	"regexp"
	"io/ioutil"
	"sort"

	"github.com/gocolly/colly"
	"sigs.k8s.io/yaml"
)

var (
	meetupRegex = regexp.MustCompile(`https://www.meetup.com/[a-zA-Z-_/0-9\.#=&]*`)
	youtubeRegex = regexp.MustCompile(`https://youtu[a-zA-Z-_/0-9\.#=&]*`)
	docsRegex = regexp.MustCompile(`https://docs.google.com[a-zA-Z-_/0-9\.#=&]*`)

)

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	listCollector := colly.NewCollector()
	dataCollector := colly.NewCollector()

	presentations := map[string]Presentation{}

	listCollector.OnHTML(".container a[href][title]", func(e *colly.HTMLElement) {
		href, title := e.Attr("href"), e.Attr("title")
		fullURL := fmt.Sprintf("https://speakerdeck.com%s", href)
		presentations[fullURL] = Presentation{
			Title: title,
			SpeakerdeckLink: NewURL(fullURL),
		}
		dataCollector.Visit(fullURL)
	})
	listCollector.OnHTML(".next .page-link[rel='next']", func(e *colly.HTMLElement) {
		fullURL := fmt.Sprintf("https://speakerdeck.com%s", e.Attr("href"))
		listCollector.Visit(fullURL)
	})

	dataCollector.OnHTML(".col-auto.text-muted", func(e *colly.HTMLElement) {
		u, date := e.Request.URL.String(), e.Text
		date = strings.Trim(strings.ReplaceAll(strings.ReplaceAll(date, ",", ""), "\n", ""), " ")
		d, _ := time.Parse("January 02 2006", date)
		p := presentations[u]
		p.Date = d
		presentations[u] = p
	})
	dataCollector.OnHTML(".deck-description.mb-4 p", func(e *colly.HTMLElement) {
		u, text := e.Request.URL.String(), e.Text
		p := presentations[u]
		y := youtubeRegex.FindStringSubmatch(text)
		if len(y) > 0 {
			p.Recording = NewURL(y[0])
		}

		d := docsRegex.FindStringSubmatch(text)
		if len(d) > 0 {
			p.PresentationLink = NewURL(d[0])
		}

		m := meetupRegex.FindStringSubmatch(text)
		if len(m) > 0 {
			p.MeetupLink = NewURL(m[0])
		}

		presentations[u] = p
	})

	listCollector.OnRequest(func(r *colly.Request) {
		fmt.Println("ListCollector visiting", r.URL)
	})
	dataCollector.OnRequest(func(r *colly.Request) {
		fmt.Println("DataCollector visiting", r.URL)
	})

	listCollector.Visit("https://speakerdeck.com/luxas")
	
	pAll := make(Presentations, 0, len(presentations))
	for _, p := range presentations {
		pAll = append(pAll, p)
	}
	sort.Sort(pAll)

	b, err := yaml.Marshal(pAll)
	if err != nil {
		return err
	}
	return ioutil.WriteFile("presentations.yaml", b, 0644)
}