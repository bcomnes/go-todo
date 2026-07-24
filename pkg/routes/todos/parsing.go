package todos

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	todostore "github.com/bcomnes/go-todo/pkg/todos"
)

var (
	errInvalidTodoID = errors.New("todo ID must be a positive integer")
	errInvalidLimit  = fmt.Errorf("limit must be a positive integer no greater than %d", todostore.MaxListLimit)
	errInvalidOffset = errors.New("offset must be a non-negative integer")
)

type pagination struct {
	Limit  int
	Offset int
}

func parsePagination(values url.Values) (pagination, error) {
	result := pagination{Limit: defaultListLimit}
	if raw, ok := singleQueryValue(values, "limit"); ok {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > todostore.MaxListLimit {
			return pagination{}, errInvalidLimit
		}
		result.Limit = limit
	} else if len(values["limit"]) > 1 {
		return pagination{}, errInvalidLimit
	}
	if raw, ok := singleQueryValue(values, "offset"); ok {
		offset, err := strconv.Atoi(raw)
		if err != nil || offset < 0 {
			return pagination{}, errInvalidOffset
		}
		result.Offset = offset
	} else if len(values["offset"]) > 1 {
		return pagination{}, errInvalidOffset
	}
	return result, nil
}

func singleQueryValue(values url.Values, key string) (string, bool) {
	entries, exists := values[key]
	returnValue := ""
	if len(entries) == 1 {
		returnValue = entries[0]
	}
	return returnValue, exists && len(entries) == 1
}

func parseTodoID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, errInvalidTodoID
	}
	return id, nil
}

// nullableString preserves the distinction between an omitted JSON field and
// an explicit null, which is required to clear an existing note.
type nullableString struct {
	Set   bool
	Value *string
}

func (value *nullableString) UnmarshalJSON(data []byte) error {
	value.Set = true
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		value.Value = nil
		return nil
	}
	var decoded string
	if err := json.Unmarshal(data, &decoded); err != nil {
		return errors.New("note must be a string or null")
	}
	value.Value = &decoded
	return nil
}

func noteFromForm(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
