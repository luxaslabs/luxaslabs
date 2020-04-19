package types

import (
	"net/url"
	"time"
)

func NewUser() *User {
	return &User{}
}

// User represents a user on speakerdeck.com
type User struct {
	Name  string `json:"name"`
	Talks Talks  `json:"talks"`
}

func NewTalk() *Talk {
	return &Talk{
		ExtraLinks: map[string]url.URL{},
	}
}

type Talk struct {
	Title      string             `json:"title"`
	Date       time.Time          `json:"date"`
	Link       url.URL            `json:"link"`
	ExtraLinks map[string]url.URL `json:"extraLinks"`
	DataID     string             `json:"dataID"`
	Hide       bool               `json:"hide"`
	Location   *Location          `json:"location,omitempty"`
}

// Talks orders the Talk objects by time
type Talks []Talk

func (p Talks) Len() int {
	return len(p)
}

func (p Talks) Less(i, j int) bool {
	return p[i].Date.Before(p[j].Date)
}

func (p Talks) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

type Location struct {
	RequestedAddress string
	ResolvedAddress  string
	Lat              float64
	Lng              float64
}
