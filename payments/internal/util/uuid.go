package util

import (
	"log"

	"github.com/google/uuid"
)

func GenerateUUID() string {
	newUUID, err := uuid.NewRandom()
	if err != nil {
		log.Fatalf("Failed to generate UUID: %v", err)
	}
	return newUUID.String()
}
