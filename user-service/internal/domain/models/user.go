package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID                  uuid.UUID
	Email               string
	FirstName           string
	LastName            string
	Login               string
	IsAdmin1            bool
	IsAdmin2            bool
	PassHash            []byte
	CreatedAt           time.Time
	UpdatedAt           time.Time
	LastVisitAt         time.Time
	InspectionsCount    int
	ErrorFeedbacksCount int
	InspectionsPerDay   int
	InspectionsForToday int
}
