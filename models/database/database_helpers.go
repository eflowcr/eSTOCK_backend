package database

import (
	"encoding/json"
	"fmt"
	"strings"
)

type StringSliceOrCSV []string

func (s *StringSliceOrCSV) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*s = nil
		return nil
	}

	var arr []string
	if err := json.Unmarshal(b, &arr); err == nil {
		*s = cleanSlice(arr)
		return nil
	}

	var str string
	if err := json.Unmarshal(b, &str); err == nil {
		*s = splitCSV(str)
		return nil
	}

	return fmt.Errorf("cannot unmarshal %s into StringSliceOrCSV", string(b))
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func cleanSlice(in []string) []string {
	var out []string
	for _, p := range in {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
