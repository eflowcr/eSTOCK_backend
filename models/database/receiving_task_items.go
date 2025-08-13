package database

import (
	"encoding/json"
	"fmt"
	"strings"
)

type StringSliceOrCSV []string

func (s *StringSliceOrCSV) UnmarshalJSON(b []byte) error {
	// null -> slice nil
	if string(b) == "null" {
		*s = nil
		return nil
	}

	// 1) Intentar []string
	var arr []string
	if err := json.Unmarshal(b, &arr); err == nil {
		*s = cleanSlice(arr)
		return nil
	}

	// 2) Intentar string CSV
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

type ReceivingTaskItem struct {
	SKU              string           `json:"sku"`
	ExpectedQuantity int              `json:"expectedQty"`
	Location         string           `json:"location"`
	LotNumbers       StringSliceOrCSV `json:"lotNumbers" gorm:"type:jsonb"`
	SerialNumbers    StringSliceOrCSV `json:"serialNumbers" gorm:"type:jsonb"`
}
