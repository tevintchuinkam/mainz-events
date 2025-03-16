package gpt

import (
	"mainz-events/internal/calendar"
)

type GPT struct {
}

func New() *GPT {
	return &GPT{}
}

func ExtractEvent(html string) *calendar.Event {
	return nil
}
