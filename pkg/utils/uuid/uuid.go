package uuid

import (
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func NewUUIDv4() uuid.UUID {
	for {
		id, err := uuid.NewRandom()
		if err == nil {
			return id
		}
	}
}

func UUIDToBinary(id uuid.UUID) bson.Binary {
	return bson.Binary{Subtype: 0x04, Data: id[:]}
}
