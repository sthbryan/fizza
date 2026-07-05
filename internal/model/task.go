package model

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	PriorityLow    = "low"
	PriorityMedium = "medium"
	PriorityHigh   = "high"
	PriorityUrgent = "urgent"
)

const DefaultPriority = PriorityMedium

var ValidPriorities = []string{
	PriorityLow,
	PriorityMedium,
	PriorityHigh,
	PriorityUrgent,
}

type Priority struct {
	Value string
}

func NewPriority(s string) (Priority, error) {
	norm := strings.ToLower(strings.TrimSpace(s))
	if norm == "" {
		return Priority{Value: DefaultPriority}, nil
	}
	for _, p := range ValidPriorities {
		if norm == p {
			return Priority{Value: p}, nil
		}
	}
	return Priority{}, ErrInvalidPriority
}

func MustPriority(s string) Priority {
	p, err := NewPriority(s)
	if err != nil {
		panic(err)
	}
	return p
}

func (p Priority) String() string {
	if p.Value == "" {
		return DefaultPriority
	}
	return p.Value
}

func (p Priority) IsZero() bool {
	return p.Value == ""
}

func (p Priority) MarshalJSON() ([]byte, error) {
	return []byte(`"` + p.String() + `"`), nil
}

func (p *Priority) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	v, err := NewPriority(s)
	if err != nil {
		return err
	}
	*p = v
	return nil
}

var (
	ErrInvalidPriority  = errors.New("priority must be one of: low, medium, high, urgent")
	ErrTitleEmpty       = errors.New("task title cannot be empty")
	ErrTaskNoBoard      = errors.New("task must belong to a board")
	ErrTaskNoColumn     = errors.New("task must belong to a column")
	ErrProjectNameEmpty = errors.New("project name cannot be empty")
	ErrProjectNameLong  = errors.New("project name too long (max 64 chars)")
	ErrProjectDescLong  = errors.New("project description too long (max 1024 chars)")
	ErrBoardNameEmpty   = errors.New("board name cannot be empty")
	ErrBoardNameLong    = errors.New("board name too long (max 64 chars)")
	ErrColumnNameEmpty  = errors.New("column name cannot be empty")
	ErrColumnNameLong   = errors.New("column name too long (max 32 chars)")
	ErrTaskCycle        = errors.New("task parent would create a cycle")
)

type Task struct {
	ID          int64      `json:"id"`
	BoardID     int64      `json:"board_id"`
	ParentID    *int64     `json:"parent_id,omitempty"`
	ColumnID    int64      `json:"column_id"`
	ColumnName  string     `json:"status"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Priority    Priority   `json:"priority"`
	Position    float64    `json:"position"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type Subtasks struct {
	Parent *Task   `json:"parent"`
	Subs   []*Task `json:"subtasks"`
}

func (t *Task) Validate() error {
	if strings.TrimSpace(t.Title) == "" {
		return ErrTitleEmpty
	}
	if t.Priority.IsZero() {
		t.Priority = Priority{Value: DefaultPriority}
	}
	if _, err := NewPriority(t.Priority.String()); err != nil {
		return err
	}
	if t.BoardID == 0 {
		return ErrTaskNoBoard
	}
	if t.ColumnID == 0 {
		return ErrTaskNoColumn
	}
	return nil
}

func ValidateProject(name, description string) error {
	if strings.TrimSpace(name) == "" {
		return ErrProjectNameEmpty
	}
	if len(name) > 64 {
		return ErrProjectNameLong
	}
	if len(description) > 1024 {
		return ErrProjectDescLong
	}
	return nil
}

func ValidateBoard(name string) error {
	if strings.TrimSpace(name) == "" {
		return ErrBoardNameEmpty
	}
	if len(name) > 64 {
		return ErrBoardNameLong
	}
	return nil
}

func ValidateColumn(name string) error {
	if strings.TrimSpace(name) == "" {
		return ErrColumnNameEmpty
	}
	if len(name) > 32 {
		return ErrColumnNameLong
	}
	return nil
}

func WrapValidation(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("validation: %w", err)
}