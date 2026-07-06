package model

import (
	"github.com/fizza/fizza/internal/dbutil"
)

type Board struct {
	ID        int64       `json:"id" db:"id"`
	ProjectID int64       `json:"project_id" db:"project_id"`
	Name      string      `json:"name" db:"name"`
	IsDefault bool        `json:"is_default" db:"is_default"`
	CreatedAt dbutil.Time `json:"created_at" db:"created_at"`
}

type Column struct {
	ID       int64  `json:"id" db:"id"`
	BoardID  int64  `json:"board_id" db:"board_id"`
	Name     string `json:"name" db:"name"`
	Position int    `json:"position" db:"position"`
	Color    string `json:"color,omitempty" db:"color"`
	WIPLimit *int   `json:"wip_limit,omitempty" db:"wip_limit"`
}

type TaskCount struct {
	ColumnID int64 `json:"column_id"`
	Count    int64 `json:"count"`
}
