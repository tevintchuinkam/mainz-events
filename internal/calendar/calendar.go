package calendar

import (
	"fmt"
	"os"

	"time"

	ical "github.com/arran4/golang-ical"
	"github.com/google/uuid"
)

type Calendar struct {
	cal *ical.Calendar
}

func New() *Calendar {
	cal := ical.NewCalendar()
	cal.SetMethod(ical.MethodRequest)
	cal.SetProductId("-//Mainz Events//Mainz Events//DE")

	return &Calendar{
		cal: cal,
	}
}

type Event struct {
	Title       string
	Summary     string
	Description string
	Location    string
	Start       time.Time
	End         time.Time
}

func (c *Calendar) AddEvent(event Event) {
	e := c.cal.AddEvent(uuid.New().String())
	e.SetCreatedTime(time.Now())
	e.SetDtStampTime(time.Now())
	e.SetModifiedAt(time.Now())
	e.SetStartAt(event.Start)
	e.SetEndAt(event.End)
	e.SetSummary(event.Title)
	e.SetLocation(event.Location)
	e.SetDescription(event.Description)
}

func (c *Calendar) Save() {
	file, err := os.Create("mainz-events.ics")
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	err = c.cal.SerializeTo(file)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}
	fmt.Println("Calendar file created successfully: mainz-events.ics")
}
