package uuid

import (
	"fmt"

	"github.com/gofrs/uuid/v5"
)

// NewUUID generates a new UUID v7 (time-ordered)
func NewUUID() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("failed to generate UUID v7: %w", err)
	}
	return id.String(), nil
}

// MustNewUUID generates a new UUID v7 or panics
func MustNewUUID() string {
	id, err := NewUUID()
	if err != nil {
		panic(fmt.Sprintf("failed to generate UUID v7: %v", err))
	}
	return id
}

// ValidateUUID checks if a string is a valid UUID format
func ValidateUUID(id string) error {
	if id == "" {
		return fmt.Errorf("UUID cannot be empty")
	}
	_, err := uuid.FromString(id)
	if err != nil {
		return fmt.Errorf("invalid UUID format: %w", err)
	}
	return nil
}

// IsEmptyUUID checks if a UUID string is empty
func IsEmptyUUID(id string) bool {
	return id == ""
}

// UUIDsToStrings converts a slice of UUID strings to strings (no-op, but kept for API consistency)
func UUIDsToStrings(ids []string) []string {
	return ids
}

// StringsToUUIDs validates and filters a slice of strings to only valid UUIDs
func StringsToUUIDs(strs []string) []string {
	result := make([]string, 0, len(strs))
	for _, str := range strs {
		if err := ValidateUUID(str); err == nil {
			result = append(result, str)
		}
	}
	return result
}
