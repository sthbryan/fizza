package model

import (
	"errors"
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

func ParsePriority(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return DefaultPriority, nil
	}
	for _, p := range ValidPriorities {
		if s == p {
			return p, nil
		}
	}
	return "", errors.New("priority must be one of: low, medium, high, urgent")
}

type Task struct {
	ID          int64      `json:"id"`
	BoardID     int64      `json:"board_id"`
	ParentID    *int64     `json:"parent_id,omitempty"`
	ColumnID    int64      `json:"column_id"`
	ColumnName  string     `json:"status"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Priority    string     `json:"priority"`
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
		return errors.New("task title cannot be empty")
	}
	if _, err := ParsePriority(t.Priority); err != nil {
		return err
	}
	if t.BoardID == 0 {
		return errors.New("task must belong to a board")
	}
	if t.ColumnID == 0 {
		return errors.New("task must belong to a column")
	}
	return nil
}

func ValidateProject(name, description string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("project name cannot be empty")
	}
	if len(name) > 64 {
		return errors.New("project name too long (max 64 chars)")
	}
	if len(description) > 1024 {
		return errors.New("project description too long (max 1024 chars)")
	}
	return nil
}

func ValidateBoard(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("board name cannot be empty")
	}
	if len(name) > 64 {
		return errors.New("board name too long (max 64 chars)")
	}
	return nil
}

func ValidateColumn(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("column name cannot be empty")
	}
	if len(name) > 32 {
		return errors.New("column name too long (max 32 chars)")
	}
	return nil
}