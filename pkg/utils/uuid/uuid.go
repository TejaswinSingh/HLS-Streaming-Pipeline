package uuid

import "github.com/google/uuid"

func NewUUIDv4() uuid.UUID {
	for {
		id, err := uuid.NewRandom()
		if err == nil {
			return id
		}
	}
}
