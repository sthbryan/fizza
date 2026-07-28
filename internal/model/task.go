package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/fizza/fizza/internal/dbutil"
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

func (p *Priority) Scan(value any) error {
	if value == nil {
		p.Value = DefaultPriority
		return nil
	}
	s, ok := value.(string)
	if !ok {
		return fmt.Errorf("Priority: cannot scan %T", value)
	}
	v, err := NewPriority(s)
	if err != nil {
		return err
	}
	*p = v
	return nil
}

var (
	ErrValidation       = errors.New("validation")
	ErrInvalidPriority  = ValidationError("priority must be one of: low, medium, high, urgent")
	ErrTitleEmpty       = ValidationError("task title cannot be empty")
	ErrTaskNoBoard      = ValidationError("task must belong to a board")
	ErrTaskNoColumn     = ValidationError("task must belong to a column")
	ErrProjectNameEmpty = ValidationError("project name cannot be empty")
	ErrProjectNameLong  = ValidationError("project name too long (max 64 chars)")
	ErrProjectDescLong  = ValidationError("project description too long (max 1024 chars)")
	ErrBoardNameEmpty   = ValidationError("board name cannot be empty")
	ErrBoardNameLong    = ValidationError("board name too long (max 64 chars)")
	ErrColumnNameEmpty  = ValidationError("column name cannot be empty")
	ErrColumnNameLong   = ValidationError("column name too long (max 32 chars)")
	ErrTaskCycle        = ValidationError("task parent would create a cycle")
)

func ValidationError(msg string) error {
	return fmt.Errorf("%w: %s", ErrValidation, msg)
}

type Task struct {
	ID          int64        `json:"id" db:"id"`
	BoardID     int64        `json:"board_id" db:"board_id"`
	ParentID    *int64       `json:"parent_id,omitempty" db:"parent_id"`
	ColumnID    int64        `json:"column_id" db:"column_id"`
	ColumnName  string       `json:"status" db:"status"`
	Title       string       `json:"title" db:"title"`
	Description string       `json:"description,omitempty" db:"description"`
	Priority    Priority     `json:"priority" db:"priority"`
	Position    float64      `json:"position" db:"position"`
	DueDate     *dbutil.Time `json:"due_date,omitempty" db:"due_date"`
	CompletedAt *dbutil.Time `json:"completed_at,omitempty" db:"completed_at"`
	ArchivedAt  *dbutil.Time `json:"archived_at,omitempty" db:"archived_at"`
	Tags        []*Tag       `json:"tags,omitempty" db:"-"`
	CreatedAt   dbutil.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   dbutil.Time  `json:"updated_at" db:"updated_at"`
}

func IsTerminalColumn(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "done", "completed", "closed":
		return true
	default:
		return false
	}
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
