package hlims

import (
	"testing"
)

func TestIdentifiers(t *testing.T) {
	seenInternal := make(map[InternalID]bool)
	seenPublic := make(map[PublicID]bool)
	for range 100 {
		internalID, err := NewInternalID()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ParseInternalID(string(internalID)); err != nil {
			t.Fatalf("ParseInternalID(%q): %v", internalID, err)
		}
		if seenInternal[internalID] {
			t.Fatalf("duplicate internal ID %q", internalID)
		}
		seenInternal[internalID] = true

		publicID, err := NewPublicID()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ParsePublicID(string(publicID)); err != nil {
			t.Fatalf("ParsePublicID(%q): %v", publicID, err)
		}
		if seenPublic[publicID] {
			t.Fatalf("duplicate public ID %q", publicID)
		}
		seenPublic[publicID] = true
	}
}

func TestInvalidIdentifiers(t *testing.T) {
	for _, value := range []string{"", "not-a-uuid", "550e8400-e29b-41d4-a716-446655440000"} {
		if _, err := ParseInternalID(value); err == nil {
			t.Errorf("ParseInternalID(%q) succeeded", value)
		}
	}
	for _, value := range []string{"", "short", "ABCDEF123456", "invalid-id!!"} {
		if _, err := ParsePublicID(value); err == nil {
			t.Errorf("ParsePublicID(%q) succeeded", value)
		}
	}
}
