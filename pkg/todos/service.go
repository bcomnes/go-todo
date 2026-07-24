// Package todos implements owner-scoped persistence and validation for todo items.
package todos

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/bcomnes/go-todo/pkg/models"
)

const (
	// MaxTaskRunes bounds task text while leaving room for descriptive tasks.
	MaxTaskRunes = 500
	// MaxNoteRunes bounds optional notes to a small text-document size.
	MaxNoteRunes = 5000
	// MaxListLimit is the largest page accepted by List.
	MaxListLimit = 100
)

var (
	// ErrNotFound is returned when an owner cannot access the requested todo.
	ErrNotFound = errors.New("todo not found")
	// ErrInvalidUserID reports a missing or invalid owner identifier.
	ErrInvalidUserID = errors.New("user ID must be greater than zero")
	// ErrInvalidTodoID reports a missing or invalid todo identifier.
	ErrInvalidTodoID = errors.New("todo ID must be greater than zero")
	// ErrTaskRequired reports a task that is empty after trimming whitespace.
	ErrTaskRequired = errors.New("task is required")
	// ErrTaskTooLong reports a task over MaxTaskRunes Unicode code points.
	ErrTaskTooLong = errors.New("task is too long")
	// ErrTaskInvalidUTF8 reports task text that PostgreSQL cannot store safely.
	ErrTaskInvalidUTF8 = errors.New("task must be valid UTF-8")
	// ErrNoteTooLong reports a note over MaxNoteRunes Unicode code points.
	ErrNoteTooLong = errors.New("note is too long")
	// ErrNoteInvalidUTF8 reports note text that PostgreSQL cannot store safely.
	ErrNoteInvalidUTF8 = errors.New("note must be valid UTF-8")
	// ErrEmptyUpdate reports a partial update with no supplied fields.
	ErrEmptyUpdate = errors.New("at least one todo field must be provided")
	// ErrInvalidLimit reports a page size outside the supported range.
	ErrInvalidLimit = errors.New("limit must be between 1 and 100")
	// ErrInvalidOffset reports a negative page offset.
	ErrInvalidOffset = errors.New("offset must be zero or greater")
)

// CreateInput contains fields accepted when creating a todo.
type CreateInput struct {
	Task string
	Done bool
	Note *string
}

// UpdateInput describes a partial todo update. NoteSet distinguishes an omitted
// note from an explicit nil note, which clears the stored value.
type UpdateInput struct {
	Task    *string
	Done    *bool
	Note    *string
	NoteSet bool
}

// Service owns todo operations for a migrated PostgreSQL database.
type Service struct {
	db *sql.DB
}

// New creates a todo service backed by db.
func New(db *sql.DB) (*Service, error) {
	if db == nil {
		return nil, errors.New("database is required")
	}
	return &Service{db: db}, nil
}

// IsValidationError reports whether err is one of the stable input-validation errors.
func IsValidationError(err error) bool {
	return errors.Is(err, ErrInvalidUserID) ||
		errors.Is(err, ErrInvalidTodoID) ||
		errors.Is(err, ErrTaskRequired) ||
		errors.Is(err, ErrTaskTooLong) ||
		errors.Is(err, ErrTaskInvalidUTF8) ||
		errors.Is(err, ErrNoteTooLong) ||
		errors.Is(err, ErrNoteInvalidUTF8) ||
		errors.Is(err, ErrEmptyUpdate) ||
		errors.Is(err, ErrInvalidLimit) ||
		errors.Is(err, ErrInvalidOffset)
}

// List returns one page of todos belonging to userID, newest first.
func (service *Service) List(ctx context.Context, userID int64, limit, offset int) ([]models.Todo, error) {
	if err := validateOwner(userID); err != nil {
		return nil, err
	}
	if limit < 1 || limit > MaxListLimit {
		return nil, ErrInvalidLimit
	}
	if offset < 0 {
		return nil, ErrInvalidOffset
	}

	rows, err := service.db.QueryContext(ctx, `
		SELECT id, user_id, task, done, note, created_at, updated_at
		FROM public.todos
		WHERE user_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list todos: %w", err)
	}
	defer rows.Close()

	result := make([]models.Todo, 0)
	for rows.Next() {
		todo, err := scanTodo(rows)
		if err != nil {
			return nil, fmt.Errorf("scan listed todo: %w", err)
		}
		result = append(result, todo)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate todos: %w", err)
	}
	return result, nil
}

// Create validates and inserts a todo owned by userID.
func (service *Service) Create(ctx context.Context, userID int64, input CreateInput) (models.Todo, error) {
	if err := validateOwner(userID); err != nil {
		return models.Todo{}, err
	}
	normalized, err := normalizeCreate(input)
	if err != nil {
		return models.Todo{}, err
	}

	todo, err := scanTodo(service.db.QueryRowContext(ctx, `
		INSERT INTO public.todos (user_id, task, done, note)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, task, done, note, created_at, updated_at
	`, userID, normalized.Task, normalized.Done, normalized.Note))
	if err != nil {
		return models.Todo{}, fmt.Errorf("insert todo: %w", err)
	}
	return todo, nil
}

// Get returns a single todo only when it belongs to userID.
func (service *Service) Get(ctx context.Context, userID, todoID int64) (models.Todo, error) {
	if err := validateIDs(userID, todoID); err != nil {
		return models.Todo{}, err
	}

	todo, err := scanTodo(service.db.QueryRowContext(ctx, `
		SELECT id, user_id, task, done, note, created_at, updated_at
		FROM public.todos
		WHERE id = $1 AND user_id = $2
	`, todoID, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return models.Todo{}, ErrNotFound
	}
	if err != nil {
		return models.Todo{}, fmt.Errorf("get todo: %w", err)
	}
	return todo, nil
}

// Update applies supplied fields and returns the updated owner-scoped record.
func (service *Service) Update(ctx context.Context, userID, todoID int64, input UpdateInput) (models.Todo, error) {
	if err := validateIDs(userID, todoID); err != nil {
		return models.Todo{}, err
	}
	normalized, err := normalizeUpdate(input)
	if err != nil {
		return models.Todo{}, err
	}

	var task string
	if normalized.Task != nil {
		task = *normalized.Task
	}
	var done bool
	if normalized.Done != nil {
		done = *normalized.Done
	}
	todo, err := scanTodo(service.db.QueryRowContext(ctx, `
		UPDATE public.todos
		SET task = CASE WHEN $3::boolean THEN $4 ELSE task END,
			done = CASE WHEN $5::boolean THEN $6 ELSE done END,
			note = CASE WHEN $7::boolean THEN $8::text ELSE note END
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, task, done, note, created_at, updated_at
	`, todoID, userID, normalized.Task != nil, task, normalized.Done != nil, done, normalized.NoteSet, normalized.Note))
	if errors.Is(err, sql.ErrNoRows) {
		return models.Todo{}, ErrNotFound
	}
	if err != nil {
		return models.Todo{}, fmt.Errorf("update todo: %w", err)
	}
	return todo, nil
}

// Toggle atomically reverses completion for an owner-scoped todo.
func (service *Service) Toggle(ctx context.Context, userID, todoID int64) (models.Todo, error) {
	if err := validateIDs(userID, todoID); err != nil {
		return models.Todo{}, err
	}

	todo, err := scanTodo(service.db.QueryRowContext(ctx, `
		UPDATE public.todos
		SET done = NOT done
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, task, done, note, created_at, updated_at
	`, todoID, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return models.Todo{}, ErrNotFound
	}
	if err != nil {
		return models.Todo{}, fmt.Errorf("toggle todo: %w", err)
	}
	return todo, nil
}

// Delete removes an owner-scoped todo.
func (service *Service) Delete(ctx context.Context, userID, todoID int64) error {
	if err := validateIDs(userID, todoID); err != nil {
		return err
	}

	var deletedID int64
	err := service.db.QueryRowContext(ctx, `
		DELETE FROM public.todos
		WHERE id = $1 AND user_id = $2
		RETURNING id
	`, todoID, userID).Scan(&deletedID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("delete todo: %w", err)
	}
	return nil
}

func validateOwner(userID int64) error {
	if userID <= 0 {
		return ErrInvalidUserID
	}
	return nil
}

func validateIDs(userID, todoID int64) error {
	if err := validateOwner(userID); err != nil {
		return err
	}
	if todoID <= 0 {
		return ErrInvalidTodoID
	}
	return nil
}

func normalizeCreate(input CreateInput) (CreateInput, error) {
	task, err := normalizeTask(input.Task)
	if err != nil {
		return CreateInput{}, err
	}
	if err := validateNote(input.Note); err != nil {
		return CreateInput{}, err
	}
	input.Task = task
	return input, nil
}

func normalizeUpdate(input UpdateInput) (UpdateInput, error) {
	if input.Task == nil && input.Done == nil && !input.NoteSet {
		return UpdateInput{}, ErrEmptyUpdate
	}
	if input.Task != nil {
		task, err := normalizeTask(*input.Task)
		if err != nil {
			return UpdateInput{}, err
		}
		input.Task = &task
	}
	if input.NoteSet {
		if err := validateNote(input.Note); err != nil {
			return UpdateInput{}, err
		}
	}
	return input, nil
}

func normalizeTask(task string) (string, error) {
	task = strings.TrimSpace(task)
	if task == "" {
		return "", ErrTaskRequired
	}
	if !utf8.ValidString(task) {
		return "", ErrTaskInvalidUTF8
	}
	if utf8.RuneCountInString(task) > MaxTaskRunes {
		return "", ErrTaskTooLong
	}
	return task, nil
}

func validateNote(note *string) error {
	if note == nil {
		return nil
	}
	if !utf8.ValidString(*note) {
		return ErrNoteInvalidUTF8
	}
	if utf8.RuneCountInString(*note) > MaxNoteRunes {
		return ErrNoteTooLong
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanTodo(source scanner) (models.Todo, error) {
	var (
		todo models.Todo
		note sql.NullString
	)
	if err := source.Scan(
		&todo.ID,
		&todo.UserID,
		&todo.Task,
		&todo.Done,
		&note,
		&todo.CreatedAt,
		&todo.UpdatedAt,
	); err != nil {
		return models.Todo{}, err
	}
	if note.Valid {
		todo.Note = &note.String
	}
	return todo, nil
}
