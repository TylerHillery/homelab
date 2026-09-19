package hlims

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	nanoid "github.com/matoous/go-nanoid/v2"
)

const (
	publicIDAlphabet = "0123456789abcdefghijklmnopqrstuvwxyz"
	publicIDLength   = 12
)

// InternalID is an application-generated UUIDv7 used for database relationships.
type InternalID string

// PublicID is a stable, externally visible NanoID.
type PublicID string

// NewInternalID generates a UUIDv7 internal ID.
func NewInternalID() (InternalID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("generate UUIDv7: %w", err)
	}
	return InternalID(id.String()), nil
}

// ParseInternalID validates and returns a UUIDv7 internal ID.
func ParseInternalID(value string) (InternalID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return "", fmt.Errorf("parse internal ID: %w", err)
	}
	if id.Version() != 7 {
		return "", fmt.Errorf("internal ID must be UUIDv7")
	}
	return InternalID(id.String()), nil
}

// NewPublicID generates a lowercase, 12-character NanoID.
func NewPublicID() (PublicID, error) {
	id, err := nanoid.Generate(publicIDAlphabet, publicIDLength)
	if err != nil {
		return "", fmt.Errorf("generate public ID: %w", err)
	}
	return PublicID(id), nil
}

// ParsePublicID validates and returns a public ID.
func ParsePublicID(value string) (PublicID, error) {
	if len(value) != publicIDLength {
		return "", fmt.Errorf("public ID must be %d characters", publicIDLength)
	}
	if strings.Trim(value, publicIDAlphabet) != "" {
		return "", fmt.Errorf("public ID contains invalid characters")
	}
	return PublicID(value), nil
}
