package core_test

import (
	"testing"
	"time"

	"github.com/bhaskarjha-com/niyantra/internal/core"
)

func TestULIDUniqueness(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := core.NewULID()
		if ids[id] {
			t.Fatalf("duplicate ULID generated: %s", id)
		}
		ids[id] = true
	}
}

func TestULIDSortability(t *testing.T) {
	now := time.Now()
	id1 := core.NewULIDAt(now)
	id2 := core.NewULIDAt(now.Add(time.Millisecond))

	if id1 >= id2 {
		t.Errorf("expected id1 < id2, got id1=%s, id2=%s", id1, id2)
	}
}

func TestULIDTimeExtraction(t *testing.T) {
	now := time.Now().Truncate(time.Millisecond)
	id := core.NewULIDAt(now)
	extracted := core.ULIDTime(id)

	if !extracted.Equal(now) {
		t.Errorf("expected extracted time to equal now, got now=%v, extracted=%v", now, extracted)
	}
}

func TestULIDFormat(t *testing.T) {
	id := core.NewULID()
	if len(id) != 26 {
		t.Errorf("expected length 26, got %d for %s", len(id), id)
	}

	validChars := "0123456789ABCDEFGHJKMNPQRSTVWXYZabcdefghjkmnpqrstvwxyz"
	for i := 0; i < len(id); i++ {
		char := id[i]
		found := false
		for j := 0; j < len(validChars); j++ {
			if char == validChars[j] {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("invalid character '%c' in ULID %s", char, id)
		}
	}
}
