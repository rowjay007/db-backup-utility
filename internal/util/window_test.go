package util

import (
	"testing"
	"time"
)

func TestInWindowSameDay(t *testing.T) {
	now := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	ok, err := InWindow(now, "09:00", "11:00", "UTC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected to be in window")
	}
}

func TestInWindowWrap(t *testing.T) {
	now := time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC)
	ok, err := InWindow(now, "23:00", "02:00", "UTC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected to be in window")
	}
}
