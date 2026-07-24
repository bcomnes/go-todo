package todos

import (
	"encoding/json"
	"errors"
	"net/url"
	"testing"
)

func TestParsePagination(t *testing.T) {
	tests := []struct {
		name    string
		values  url.Values
		want    pagination
		wantErr error
	}{
		{name: "defaults", values: url.Values{}, want: pagination{Limit: 20, Offset: 0}},
		{name: "explicit", values: url.Values{"limit": {"100"}, "offset": {"40"}}, want: pagination{Limit: 100, Offset: 40}},
		{name: "zero limit", values: url.Values{"limit": {"0"}}, wantErr: errInvalidLimit},
		{name: "large limit", values: url.Values{"limit": {"101"}}, wantErr: errInvalidLimit},
		{name: "non-number limit", values: url.Values{"limit": {"many"}}, wantErr: errInvalidLimit},
		{name: "duplicate limit", values: url.Values{"limit": {"10", "20"}}, wantErr: errInvalidLimit},
		{name: "negative offset", values: url.Values{"offset": {"-1"}}, wantErr: errInvalidOffset},
		{name: "duplicate offset", values: url.Values{"offset": {"0", "20"}}, wantErr: errInvalidOffset},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parsePagination(test.values)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("error = %v, want %v", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parsePagination: %v", err)
			}
			if got != test.want {
				t.Fatalf("pagination = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestParseTodoID(t *testing.T) {
	if id, err := parseTodoID("42"); err != nil || id != 42 {
		t.Fatalf("parseTodoID(42) = %d, %v", id, err)
	}
	for _, raw := range []string{"", "0", "-1", "1.5", "nope"} {
		if _, err := parseTodoID(raw); !errors.Is(err, errInvalidTodoID) {
			t.Fatalf("parseTodoID(%q) error = %v, want %v", raw, err, errInvalidTodoID)
		}
	}
}

func TestUpdateAPIRequestDistinguishesOmittedNullAndStringNote(t *testing.T) {
	var omitted updateAPIRequest
	if err := json.Unmarshal([]byte(`{"done":true}`), &omitted); err != nil {
		t.Fatalf("decode omitted note: %v", err)
	}
	if omitted.Note.Set {
		t.Fatal("omitted note was marked set")
	}

	var cleared updateAPIRequest
	if err := json.Unmarshal([]byte(`{"note":null}`), &cleared); err != nil {
		t.Fatalf("decode null note: %v", err)
	}
	if !cleared.Note.Set || cleared.Note.Value != nil {
		t.Fatalf("cleared note = %#v", cleared.Note)
	}

	var changed updateAPIRequest
	if err := json.Unmarshal([]byte(`{"note":"details"}`), &changed); err != nil {
		t.Fatalf("decode string note: %v", err)
	}
	if !changed.Note.Set || changed.Note.Value == nil || *changed.Note.Value != "details" {
		t.Fatalf("changed note = %#v", changed.Note)
	}

	var invalid updateAPIRequest
	if err := json.Unmarshal([]byte(`{"note":false}`), &invalid); err == nil {
		t.Fatal("non-string note decoded without error")
	}
}

func TestNoteFromFormUsesEmptyStringAsNull(t *testing.T) {
	if note := noteFromForm(""); note != nil {
		t.Fatalf("empty note = %q, want nil", *note)
	}
	if note := noteFromForm("details"); note == nil || *note != "details" {
		t.Fatalf("note = %#v, want details", note)
	}
}
