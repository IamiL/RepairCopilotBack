package chatmodel

import (
	"time"

	"github.com/google/uuid"
)

type Chat struct {
	Id            uuid.UUID
	UserID        uuid.UUID
	CreatedAt     time.Time
	MessagesCount int
	IsFinished    bool
	Conclusion    string
	IsProcessing  bool
	Enclosure     int
}
