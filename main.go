package main

import (
	"fmt"
	"log"
	"mainz-events/internal/calendar"
	"mainz-events/internal/scraper"
)

func main() {
	cal := calendar.New()

	defer func() {
		if r := recover(); r != nil {
			log.Fatalln(r)
		}
		cal.Save()
	}()

	previews := scraper.GetAllEventPreviewsParallel()
	for _, preview := range previews {

		if preview.Title != "" {
			event := calendar.Event{
				Title:       preview.Title,
				Start:       preview.StartTime,
				End:         preview.EndTime,
				Location:    preview.Location,
				Description: preview.Link,
			}
			cal.AddEvent(event)
			fmt.Println("added event", event)
		}
	}
}
