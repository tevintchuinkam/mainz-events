package main

import (
	"fmt"
	"log"
	"mainz-events/internal/calendar"
	"mainz-events/internal/scraper"
)

func main() {
	iterator := scraper.NewEventLinkIterator()
	cal := calendar.New()

	defer func() {
		if r := recover(); r != nil {
			log.Fatalln(r)
		}
		cal.Save()
	}()

	for {
		link, hasMore := iterator.Next()
		if !hasMore {
			break
		}
		event, err := scraper.ScrapeEvent(link)
		if err != nil {
			log.Fatalln(err)
			continue
		}
		if event != nil && event.Title != "" {
			fmt.Println("added event", event)
			cal.AddEvent(*event)
		}
	}
}
