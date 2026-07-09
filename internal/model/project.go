package model

import (
	"github.com/fizza/fizza/internal/dbutil"
)

type Project struct {
	ID          int64       `json:"id" db:"id"`
	Name        string      `json:"name" db:"name"`
	Description string      `json:"description,omitempty" db:"description"`
	BoardCount  int64       `json:"board_count" db:"board_count"`
	CreatedAt   dbutil.Time `json:"created_at" db:"created_at"`
	UpdatedAt   dbutil.Time `json:"updated_at" db:"updated_at"`
}

type ProjectCounts struct {
	Projects int64 `json:"projects"`
	Boards   int64 `json:"boards"`
	Tasks    int64 `json:"tasks"`
}

type Event struct {
	ID        int64       `json:"id" db:"id"`
	ProjectID *int64      `json:"project_id,omitempty" db:"project_id"`
	BoardID   *int64      `json:"board_id,omitempty" db:"board_id"`
	TaskID    *int64      `json:"task_id,omitempty" db:"task_id"`
	Kind      string      `json:"kind" db:"kind"`
	Payload   string      `json:"payload,omitempty" db:"payload"`
	CreatedAt dbutil.Time `json:"created_at" db:"created_at"`
}
