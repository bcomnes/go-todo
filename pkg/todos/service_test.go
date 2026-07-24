package todos

import (
	"database/sql"
	"errors"
	"strings"
	"testing"
)

func TestNewRequiresDatabase(t *testing.T) {
	if _, err := New(nil); err == nil {
		t.Fatal("New(nil) returned no error")
	}
}

func TestNormalizeCreateTrimsAndValidatesTask(t *testing.T) {
	input, err := normalizeCreate(CreateInput{Task: "  ship it  "})
	if err != nil {
		t.Fatalf("normalizeCreate: %v", err)
	}
	if input.Task != "ship it" {
		t.Fatalf("task = %q, want %q", input.Task, "ship it")
	}

	for _, task := range []string{"", " \n\t "} {
		_, err := normalizeCreate(CreateInput{Task: task})
		if !errors.Is(err, ErrTaskRequired) {
			t.Fatalf("normalizeCreate(%q) error = %v, want %v", task, err, ErrTaskRequired)
		}
	}
}

func TestNormalizeCreateBoundsUnicodeText(t *testing.T) {
	_, err := normalizeCreate(CreateInput{Task: strings.Repeat("界", MaxTaskRunes+1)})
	if !errors.Is(err, ErrTaskTooLong) {
		t.Fatalf("long task error = %v, want %v", err, ErrTaskTooLong)
	}

	note := strings.Repeat("界", MaxNoteRunes+1)
	_, err = normalizeCreate(CreateInput{Task: "task", Note: &note})
	if !errors.Is(err, ErrNoteTooLong) {
		t.Fatalf("long note error = %v, want %v", err, ErrNoteTooLong)
	}
}

func TestNormalizeUpdateSupportsClearingNoteAndRejectsEmptyPatch(t *testing.T) {
	if _, err := normalizeUpdate(UpdateInput{}); !errors.Is(err, ErrEmptyUpdate) {
		t.Fatalf("empty update error = %v, want %v", err, ErrEmptyUpdate)
	}

	input, err := normalizeUpdate(UpdateInput{NoteSet: true, Note: nil})
	if err != nil {
		t.Fatalf("clear note update: %v", err)
	}
	if !input.NoteSet || input.Note != nil {
		t.Fatalf("clear note update = %#v", input)
	}
}

func TestPublicMethodsValidateBeforeUsingDatabase(t *testing.T) {
	service, err := New(&sql.DB{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := service.List(t.Context(), 1, MaxListLimit+1, 0); !errors.Is(err, ErrInvalidLimit) {
		t.Fatalf("List invalid limit error = %v, want %v", err, ErrInvalidLimit)
	}
	if _, err := service.List(t.Context(), 1, 20, -1); !errors.Is(err, ErrInvalidOffset) {
		t.Fatalf("List invalid offset error = %v, want %v", err, ErrInvalidOffset)
	}
	if _, err := service.Create(t.Context(), 1, CreateInput{Task: "  "}); !errors.Is(err, ErrTaskRequired) {
		t.Fatalf("Create invalid task error = %v, want %v", err, ErrTaskRequired)
	}
	if _, err := service.Update(t.Context(), 1, 1, UpdateInput{}); !errors.Is(err, ErrEmptyUpdate) {
		t.Fatalf("Update empty patch error = %v, want %v", err, ErrEmptyUpdate)
	}
	if err := service.Delete(t.Context(), 1, 0); !errors.Is(err, ErrInvalidTodoID) {
		t.Fatalf("Delete invalid ID error = %v, want %v", err, ErrInvalidTodoID)
	}
}
