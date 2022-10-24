package core

import (
	"errors"

	"github.com/google/uuid"
)

// UUID is a uuid object that yaml serializes to string format
type UUID struct {
	uuid.UUID
}

// MarshalYAML implements the json.Marshaler interface.
func (t UUID) MarshalYAML() (interface{}, error) {
	return t.UUID.String(), nil
}

// UnmarshalYAML implements the json.Unmarshaler interface.
func (t UUID) UnmarshalYAML(data []byte) error {
	parsedUUID, err := uuid.Parse(string(data))
	if err != nil {
		return err
	}
	t.UUID = parsedUUID
	return nil
}

// UUIDFromBytes returns a uuid and panics on err. Use wisely.
func UUIDFromBytes(data []byte) (uuid uuid.UUID) {
	if len(data) != 16 {
		panic(errors.New("wrong length"))
	}
	err := uuid.UnmarshalBinary(data)
	if err != nil {
		panic(err)
	}
	return uuid
}
