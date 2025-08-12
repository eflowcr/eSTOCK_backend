package tools

import (
	"github.com/google/uuid"
)

func GenerateGUID() string {
	return uuid.New().String()
}
