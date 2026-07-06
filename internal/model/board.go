package model

import "time"

type Board struct {
	ID        int64     `json:"id"`
	ProjectID int64     `json:"project_id"`
	Name      string    `json:"name"`
	IsDefault bool      `json:"is_default"`
	CreatedAt time.Time `json:"created_at"`
}

type Column struct {
	ID       int64  `json:"id"`
	BoardID  int64  `json:"board_id"`
	Name     string `json:"name"`
	Position int    `json:"position"`
	Color    string `json:"color,omitempty"`
	WIPLimit *int   `json:"wip_limit,omitempty"`
}

type TaskCount struct {
	ColumnID int64 `json:"column_id"`
	Count    int64 `json:"count"`
}