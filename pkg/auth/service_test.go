package auth

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
)

func TestHashCapacityRejectsWaiters(t *testing.T) {
	service, err := New(&sql.DB{}, time.Hour, Options{HashConcurrency: 1})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := service.acquireHashSlot(context.Background()); err != nil {
		t.Fatalf("acquire first hash slot: %v", err)
	}
	defer service.releaseHashSlot()

	if err := service.acquireHashSlot(context.Background()); !errors.Is(err, ErrCapacity) {
		t.Fatalf("second acquisition error = %v, want %v", err, ErrCapacity)
	}
}
