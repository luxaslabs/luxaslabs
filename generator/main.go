package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"

	"github.com/luxaslabs/luxaslabs/generator/speakerdeck"
	"sigs.k8s.io/yaml"
)

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	in, err := ioutil.ReadFile("data.yaml")
	if err != nil {
		return err
	}

	site := Site{}
	if err := yaml.Unmarshal(in, &site); err != nil {
		return err
	}

	scraper := speakerdeck.NewUserPageScraper()

	for i, _ := range site.Persons {

		sdUser, err := scraper.Scrape("luxas")
		if err != nil {
			return err
		}

		for _, talk := range sdUser.Talks {
			p := Presentation{}
			p.Title = talk.Title
			p.Date = talk.Date
			p.SpeakerdeckLink = NewURL(talk.Link.String())
			for domain, extraLink := range talk.ExtraLinks {
				if strings.Contains(domain, "meetup.com") {
					p.MeetupLink = NewURL(extraLink.String())
				} else if strings.Contains(domain, "youtu") {
					p.Recording = NewURL(extraLink.String())
				} else if strings.Contains(domain, "docs.google.com") {
					p.PresentationLink = NewURL(extraLink.String())
				}
			}
			// append to presentations
			site.Persons[i].Presentations = append(site.Persons[i].Presentations, p)
		}
		sort.Sort(site.Persons[i].Presentations)
	}

	output, err := json.MarshalIndent(site, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile("generated.json", output, 0644)
}
