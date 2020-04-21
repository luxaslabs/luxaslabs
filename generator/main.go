package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"

	"github.com/luxas/speakerdeck-api"
	"github.com/luxas/speakerdeck-api/location"
	"github.com/luxas/speakerdeck-api/scraper"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/yaml"
)

var mapsAPIKey = flag.String("maps-api-key", "", "Google Maps API key with the Geocoding API usage set")

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	flag.Parse()
	in, err := ioutil.ReadFile("data.yaml")
	if err != nil {
		return err
	}

	site := Site{}
	if err := yaml.Unmarshal(in, &site); err != nil {
		return err
	}

	// Enable geolocation if we've got an API key
	var opts *scraper.ScrapeOptions
	if len(*mapsAPIKey) != 0 {
		locext, err := location.NewLocationExtension(*mapsAPIKey)
		if err != nil {
			return err
		}
		opts = &scraper.ScrapeOptions{
			Extensions: []scraper.Extension{locext},
		}
	}

	for i, _ := range site.Persons {
		talks, err := speakerdeck.ScrapeTalks("luxas", "", opts)
		if err != nil {
			return err
		}

		for _, talk := range talks {
			if talk.Hide {
				log.Infof("Hiding presentation %s", talk.Title)
				continue
			}

			p := Presentation{}
			p.Title = talk.Title
			p.Date = talk.Date
			p.SpeakerdeckID = talk.DataID
			p.SpeakerdeckLink = NewURL(talk.Link)
			if talk.Location != nil {
				p.Location = &Location{
					Address:   talk.Location.RequestedAddress,
					Latitude:  talk.Location.Lat,
					Longitude: talk.Location.Lng,
				}
			}

			for domain, extraLinks := range talk.ExtraLinks {
				if strings.Contains(domain, "meetup.com") {
					p.MeetupLink = NewURL(extraLinks[0])
				} else if strings.Contains(domain, "youtu") {
					p.Recording = NewURL(extraLinks[0])
				} else if strings.Contains(domain, "docs.google.com") {
					p.PresentationLink = NewURL(extraLinks[0])
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
