package tools

import (
	"strconv"
	"time"
)

func StrPtr(s string) *string { return &s }

func StringToInt(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.Atoi(s)
}

func ParseDate(s string) *time.Time {
	if s == "" {
		return nil
	}
	layouts := []string{
		"2006-01-02",
		"02-01-2006",
		"2006/01/02",
		"02/01/2006",
		"2006.01.02",
		"02.01.2006",
		"2006-1-2",
		"2-1-2006",
		"2006/1/2",
		"2/1/2006",
		"2006.1.2",
		"2.1.2006",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return &t
		}
	}
	return nil
}
