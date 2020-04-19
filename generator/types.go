package main

import (
	"net/url"
	"time"
)

func NewURL(u string) *URL {
	u2, err := url.Parse(u)
	if err != nil {
		return nil
	}
	u3 := URL(u2.String())
	return &u3
}

type URL string

type Site struct {
	Persons   []Person   `json:"persons"`
	Company   Company    `json:"company"`
	BlogPosts []BlogPost `json:"blogPosts"`
}

type BlogPost struct {
	Name       string    `json:"name"`
	Authors    []string  `json:"authors"`
	Date       time.Time `json:"date"`
	SourceLink URL       `json:"sourceLink"`
}

type Company struct {
	Name      string     `json:"name"`
	ID        string     `json:"id"`
	Email     string     `json:"email"`
	Solutions []Solution `json:"solutions"`
	Partners  []Company  `json:"partners"`
	Projects  []Project  `json:"projects"`
}

type Solution struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Project struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	StartDate   time.Time `json:"startDate"`
	EndDate     time.Time `json:"endDate"`
}

type Person struct {
	Name           string          `json:"name"`
	Email          string          `json:"email`
	Biography      string          `json:"biography"`
	SocialMedia    SocialMedia     `json:"socialMedia"`
	Certifications []Certification `json:"certifications"`
	Positions      []Position      `json:"positions"`
	Presentations  Presentations   `json:"presentations"`
}

type Presentations []Presentation

func (p Presentations) Len() int {
	return len(p)
}

func (p Presentations) Less(i, j int) bool {
	return p[i].Date.Before(p[j].Date)
}

func (p Presentations) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

type Presentation struct {
	Title            string    `json:"title"`
	Date             time.Time `json:"date"`
	SpeakerdeckLink  *URL      `json:"speakerdeckLink"`
	SpeakerdeckID    string    `json:"speakerdeckID"`
	PresentationLink *URL      `json:"presentationLink,omitempty"`
	MeetupLink       *URL      `json:"meetupLink,omitempty"`
	Recording        *URL      `json:"recording,omitempty"`
	Location         *Location `json:"location,omitempty"`
}

type Location struct {
	Address   string  `json:"address"`
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
}

type SocialMedia struct {
	Github      string            `json:"github"`
	Twitter     string            `json:"twitter`
	SpeakerDeck string            `json:"speakerdeck"`
	LinkedIn    string            `json:"linkedin`
	Slack       map[string]string `json:"slack`
}

type Certification struct {
	Name        string `json:"name"`
	Short       string `json:"short"`
	Description string `json:"description"`
	Issuer      string `json:"issuer"`
	AcclaimLink *URL   `json:"acclaimLink"`
}

type Position struct {
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	Organization string     `json:"organization"`
	StartDate    *time.Time `json:"startDate"`
	EndDate      *time.Time `json:"endDate"`
}
