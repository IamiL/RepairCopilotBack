package messagemodel

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	Id        uuid.UUID
	Content   string
	ChatId    uuid.UUID
	Role      string
	CreatedAt time.Time
}
