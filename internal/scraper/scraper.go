package scraper

import (
	"fmt"
	"log"
	"mainz-events/internal/calendar"
	"net/http"
	"strings"
	"sync"
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

// EventPreview contains basic information about an event from the listing page
type EventPreview struct {
	Link      string
	Title     string
	Organizer string
	Location  string
	StartTime time.Time
	EndTime   time.Time
}

// EventPreviewIterator provides an iterator for event previews from the Mainz website
type EventPreviewIterator struct {
	currentPage int
	previews    []EventPreview
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

// GetAllEventLinksParallel collects all event links from the website in parallel
func GetAllEventLinksParallel() []string {
	var allLinks []string
	currentPage := 1
	hasMore := true
	batchSize := 10

	for hasMore {
		var wg sync.WaitGroup
		linksChan := make(chan []string, batchSize)

		// Start goroutines for fetching multiple pages concurrently
		for i := 0; i < batchSize; i++ {
			wg.Add(1)
			go func(page int) {
				defer wg.Done()

				pageURL := fmt.Sprintf("%s?sp-page=%d", baseURL, page)
				log.Printf("Fetching page %d: %s", page, pageURL)

				// Make HTTP request
				resp, err := http.Get(pageURL)
				if err != nil {
					log.Printf("Error fetching page %d: %v", page, err)
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode != 200 {
					log.Printf("Status code error on page %d: %d %s", page, resp.StatusCode, resp.Status)
					return
				}

				// Parse HTML document
				doc, err := goquery.NewDocumentFromReader(resp.Body)
				if err != nil {
					log.Printf("Error parsing HTML for page %d: %v", page, err)
					return
				}

				// Find all event links
				pageLinks := []string{}
				doc.Find("a").Each(func(i int, s *goquery.Selection) {
					href, exists := s.Attr("href")
					if exists && strings.Contains(href, linkIdentifier) {
						// Make sure we have absolute URLs
						if !strings.HasPrefix(href, "http") {
							href = "https://www.mainz.de" + href
						}
						pageLinks = append(pageLinks, href)
					}
				})

				if len(pageLinks) > 0 {
					linksChan <- pageLinks
				}
			}(currentPage + i)
		}

		// Close channel when all goroutines are done
		go func() {
			wg.Wait()
			close(linksChan)
		}()

		// Collect links from channel
		pageLinksCount := 0
		for links := range linksChan {
			allLinks = append(allLinks, links...)
			pageLinksCount++
		}

		// If we got fewer pages than requested, we're done
		if pageLinksCount < batchSize {
			hasMore = false
		}

		// Move to the next batch of pages
		currentPage += batchSize
	}

	return allLinks
}

// NewEventPreviewIterator creates a new iterator for event previews
func NewEventPreviewIterator() *EventPreviewIterator {
	return &EventPreviewIterator{
		currentPage: 1,
		previews:    []EventPreview{},
		hasMore:     true,
	}
}

// Next returns the next event preview and a boolean indicating if there are more previews
func (e *EventPreviewIterator) Next() (EventPreview, bool) {
	// If we have previews in our buffer, return the next one
	if len(e.previews) > 0 {
		preview := e.previews[0]
		e.previews = e.previews[1:]
		return preview, true
	}

	// If we don't have previews but there are no more pages, we're done
	if !e.hasMore {
		return EventPreview{}, false
	}

	// Otherwise, fetch the next page of previews
	e.fetchNextPage()

	// If we found previews, return the first one
	if len(e.previews) > 0 {
		preview := e.previews[0]
		e.previews = e.previews[1:]
		return preview, true
	}

	// No more previews
	return EventPreview{}, false
}

// fetchNextPage retrieves the next page of event previews
func (e *EventPreviewIterator) fetchNextPage() {
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

	// Find all event list items
	previews := []EventPreview{}
	doc.Find("li.SPmod-events-teaser").Each(func(i int, s *goquery.Selection) {
		preview := EventPreview{}

		// Extract link
		linkElement := s.Find("a[itemprop='url']")
		if href, exists := linkElement.Attr("href"); exists {
			// Make sure we have absolute URLs
			if !strings.HasPrefix(href, "http") {
				href = "https://www.mainz.de" + href
			}
			preview.Link = href
		}

		// Extract title
		preview.Title = strings.TrimSpace(s.Find("h3[itemprop='name']").Text())

		// Extract location/organizer
		locationText := strings.TrimSpace(s.Find(".SPmod-events-location").Text())
		preview.Location = locationText
		preview.Organizer = locationText // In many cases, these are the same

		// Extract start time
		if startDateAttr, exists := s.Find("meta[itemprop='startDate']").Attr("content"); exists {
			startTime, err := time.Parse("2006-01-02T15:04:05-0700", startDateAttr)
			if err == nil {
				preview.StartTime = startTime
			}
		}

		// Extract end time
		if endDateAttr, exists := s.Find("meta[itemprop='endDate']").Attr("content"); exists {
			endTime, err := time.Parse("2006-01-02T15:04:05-0700", endDateAttr)
			if err == nil {
				preview.EndTime = endTime
			}
		}

		previews = append(previews, preview)
	})

	// If we didn't find any previews, there are no more pages
	if len(previews) == 0 {
		e.hasMore = false
		return
	}

	// Store the previews and increment the page counter
	e.previews = previews
	e.currentPage++
}

// GetAllEventPreviews collects all event previews from the website
func GetAllEventPreviews() []EventPreview {
	iterator := NewEventPreviewIterator()
	var allPreviews []EventPreview

	for {
		preview, hasMore := iterator.Next()
		if !hasMore {
			break
		}
		allPreviews = append(allPreviews, preview)
	}

	return allPreviews
}

// GetAllEventPreviewsParallel collects all event previews from the website in parallel
func GetAllEventPreviewsParallel() []EventPreview {
	var allPreviews []EventPreview
	currentPage := 1
	hasMore := true
	batchSize := 10

	for hasMore {
		var wg sync.WaitGroup
		previewsChan := make(chan []EventPreview, batchSize)

		// Start goroutines for fetching multiple pages concurrently
		for i := 0; i < batchSize; i++ {
			wg.Add(1)
			go func(page int) {
				defer wg.Done()

				pageURL := fmt.Sprintf("%s?sp-page=%d", baseURL, page)
				log.Printf("Fetching page %d: %s", page, pageURL)

				// Make HTTP request
				resp, err := http.Get(pageURL)
				if err != nil {
					log.Printf("Error fetching page %d: %v", page, err)
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode != 200 {
					log.Printf("Status code error on page %d: %d %s", page, resp.StatusCode, resp.Status)
					return
				}

				// Parse HTML document
				doc, err := goquery.NewDocumentFromReader(resp.Body)
				if err != nil {
					log.Printf("Error parsing HTML for page %d: %v", page, err)
					return
				}

				// Find all event list items
				pagePreviews := []EventPreview{}
				doc.Find("li.SPmod-events-teaser").Each(func(i int, s *goquery.Selection) {
					preview := EventPreview{}

					// Extract link
					linkElement := s.Find("a[itemprop='url']")
					if href, exists := linkElement.Attr("href"); exists {
						// Make sure we have absolute URLs
						if !strings.HasPrefix(href, "http") {
							href = "https://www.mainz.de" + href
						}
						preview.Link = href
					}

					// Extract title
					preview.Title = strings.TrimSpace(s.Find("h3[itemprop='name']").Text())

					// Extract location/organizer
					locationText := strings.TrimSpace(s.Find(".SPmod-events-location").Text())
					preview.Location = locationText
					preview.Organizer = locationText // In many cases, these are the same

					// Extract start time
					if startDateAttr, exists := s.Find("meta[itemprop='startDate']").Attr("content"); exists {
						startTime, err := time.Parse("2006-01-02T15:04:05-0700", startDateAttr)
						if err == nil {
							log.Fatalln("start time", startTime)
							preview.StartTime = startTime
						}
					}

					// Extract end time
					if endDateAttr, exists := s.Find("meta[itemprop='endDate']").Attr("content"); exists {
						endTime, err := time.Parse("2006-01-02T15:04:05-0700", endDateAttr)
						if err == nil {
							log.Fatalln("end time", endTime)
							preview.EndTime = endTime
						}
					}

					pagePreviews = append(pagePreviews, preview)
					fmt.Println("collected page preview", preview)
				})

				if len(pagePreviews) > 0 {
					previewsChan <- pagePreviews
				}
			}(currentPage + i)
		}

		// Close channel when all goroutines are done
		go func() {
			wg.Wait()
			close(previewsChan)
		}()

		// Collect previews from channel
		pagePreviewsCount := 0
		for previews := range previewsChan {
			allPreviews = append(allPreviews, previews...)
			pagePreviewsCount++
		}

		// If we got fewer pages than requested, we're done
		if pagePreviewsCount < batchSize {
			hasMore = false
		}

		// Move to the next batch of pages
		currentPage += batchSize
	}

	return allPreviews
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
