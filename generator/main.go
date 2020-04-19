package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"

	"github.com/luxaslabs/luxaslabs/generator/speakerdeck"
	"github.com/luxaslabs/luxaslabs/generator/speakerdeck/location"
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
	in, err := ioutil.ReadFile("data.yaml")
	if err != nil {
		return err
	}

	site := Site{}
	if err := yaml.Unmarshal(in, &site); err != nil {
		return err
	}

	us := speakerdeck.NewUserScraper()

	// Enable geolocation if we've got an API key
	if len(*mapsAPIKey) != 0 {
		locext, err := location.NewLocationExtension(*mapsAPIKey)
		if err != nil {
			return err
		}
		if err := us.AddExtension(locext); err != nil {
			return err
		}
	}

	for i, _ := range site.Persons {

		sdUser, err := us.ScrapeUser("luxas")
		if err != nil {
			return err
		}

		for _, talk := range sdUser.Talks {
			if talk.Hide {
				log.Infof("Hiding presentation %s", talk.Title)
				continue
			}

			p := Presentation{}
			p.Title = talk.Title
			p.Date = talk.Date
			p.SpeakerdeckID = talk.DataID
			p.SpeakerdeckLink = NewURL(talk.Link.String())
			if talk.Location != nil {
				p.Location = &Location{
					Address:   talk.Location.RequestedAddress,
					Latitude:  talk.Location.Lat,
					Longitude: talk.Location.Lng,
				}
			}

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
