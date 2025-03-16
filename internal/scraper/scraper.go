package scraper

import (
	"fmt"
	"log"
	"mainz-events/internal/calendar"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	baseURL        = "https://www.mainz.de/freizeit-und-sport/feste-und-veranstaltungen/veranstaltungskalender.php"
	linkIdentifier = "/freizeit-und-sport/feste-und-veranstaltungen/"
)

// EventLinkIterator provides an iterator for event links from the Mainz website
type EventLinkIterator struct {
	currentPage int
	links       []string
	hasMore     bool
}

// NewEventLinkIterator creates a new iterator for event links
func NewEventLinkIterator() *EventLinkIterator {
	return &EventLinkIterator{
		currentPage: 1,
		links:       []string{},
		hasMore:     true,
	}
}

// Next returns the next event link and a boolean indicating if there are more links
func (e *EventLinkIterator) Next() (string, bool) {
	// If we have links in our buffer, return the next one
	if len(e.links) > 0 {
		link := e.links[0]
		e.links = e.links[1:]
		return link, true
	}

	// If we don't have links but there are no more pages, we're done
	if !e.hasMore {
		return "", false
	}

	// Otherwise, fetch the next page of links
	e.fetchNextPage()

	// If we found links, return the first one
	if len(e.links) > 0 {
		link := e.links[0]
		e.links = e.links[1:]
		return link, true
	}

	// No more links
	return "", false
}

// fetchNextPage retrieves the next page of event links
func (e *EventLinkIterator) fetchNextPage() {
	pageURL := fmt.Sprintf("%s?sp-page=%d", baseURL, e.currentPage)
	log.Printf("Fetching page %d: %s", e.currentPage, pageURL)

	// Make HTTP request
	resp, err := http.Get(pageURL)
	if err != nil {
		log.Printf("Error fetching page %d: %v", e.currentPage, err)
		e.hasMore = false
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("Status code error: %d %s", resp.StatusCode, resp.Status)
		e.hasMore = false
		return
	}

	// Parse HTML document
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("Error parsing HTML: %v", err)
		e.hasMore = false
		return
	}

	// Find all event links
	links := []string{}
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists && strings.Contains(href, linkIdentifier) {
			// Make sure we have absolute URLs
			if !strings.HasPrefix(href, "http") {
				href = "https://www.mainz.de" + href
			}
			links = append(links, href)
		}
	})

	// If we didn't find any links, there are no more pages
	if len(links) == 0 {
		e.hasMore = false
		return
	}

	// Store the links and increment the page counter
	e.links = links
	e.currentPage++
}

// GetAllEventLinks collects all event links from the website
func GetAllEventLinks() []string {
	iterator := NewEventLinkIterator()
	var allLinks []string

	for {
		link, hasMore := iterator.Next()
		if !hasMore {
			break
		}
		allLinks = append(allLinks, link)
	}

	return allLinks
}

// ScrapeEvent fetches and parses an event page to extract event details
func ScrapeEvent(url string) (*calendar.Event, error) {
	// Make HTTP request
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching event page: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status code error: %d %s", resp.StatusCode, resp.Status)
	}

	// Parse HTML document
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error parsing HTML: %v", err)
	}

	// Create a new event
	event := &calendar.Event{}

	// Extract event content from the article with id "SP-content"
	article := doc.Find("article#SP-content")
	if article.Length() == 0 {
		return nil, fmt.Errorf("event content not found")
	}

	// Extract title
	event.Title = strings.TrimSpace(article.Find("h1[itemprop='name']").Text())

	// Extract date
	dateAttr, exists := article.Find("time[itemprop='startDate']").Attr("datetime")
	if exists {
		// Parse the date from the datetime attribute (format: "2025-03-16")
		startDate, err := time.Parse("2006-01-02", dateAttr)
		if err == nil {
			event.Start = startDate
			// Since we don't have an end time, set end to the same day
			event.End = startDate
		}
	}

	// Extract description
	// First try to get the teaser
	teaser := strings.TrimSpace(article.Find(".event-teaser").Text())

	// Then get the main description
	description := ""
	article.Find("div[itemprop='description'] .SP-text p").Each(func(i int, s *goquery.Selection) {
		if description != "" {
			description += "\n"
		}
		description += strings.TrimSpace(s.Text())
	})

	// Combine teaser and description
	if teaser != "" {
		event.Summary = teaser
		event.Description = description
	} else {
		// If no teaser, use first part of description as summary
		if len(description) > 100 {
			event.Summary = description[:100] + "..."
			event.Description = description
		} else {
			event.Summary = description
			event.Description = description
		}
	}

	// Extract location
	locationName := strings.TrimSpace(article.Find(".SPmod-events-location .SP-contact-organisation").Text())
	locationStreet := strings.TrimSpace(article.Find(".SPmod-events-location .SP-contact-streetAddress").Text())
	locationPostal := strings.TrimSpace(article.Find(".SPmod-events-location .SP-contact-postalCode").Text())
	locationCity := strings.TrimSpace(article.Find(".SPmod-events-location .SP-contact-addressLocality").Text())

	// Combine location parts
	locationParts := []string{}
	if locationName != "" {
		locationParts = append(locationParts, locationName)
	}
	if locationStreet != "" {
		locationParts = append(locationParts, locationStreet)
	}
	if locationPostal != "" || locationCity != "" {
		locationParts = append(locationParts, strings.TrimSpace(locationPostal+" "+locationCity))
	}

	event.Location = strings.Join(locationParts, ", ")

	return event, nil
}
