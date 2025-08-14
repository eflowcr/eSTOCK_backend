package tools

import "strconv"

func StrPtr(s string) *string { return &s }

func StringToInt(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.Atoi(s)
}
