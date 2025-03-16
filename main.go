package main

import (
	"fmt"
	"mainz-events/internal/scraper"
)

func main() {
	iterator := scraper.NewEventLinkIterator()

	for {
		link, hasMore := iterator.Next()
		if !hasMore {
			break
		}
		fmt.Println(link)
	}

}
