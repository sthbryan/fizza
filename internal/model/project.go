package model

import "time"

type Project struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ProjectCounts struct {
	Projects int64 `json:"projects"`
	Boards   int64 `json:"boards"`
	Tasks    int64 `json:"tasks"`
}

type Event struct {
	ID        int64     `json:"id"`
	ProjectID *int64    `json:"project_id,omitempty"`
	BoardID   *int64    `json:"board_id,omitempty"`
	TaskID    *int64    `json:"task_id,omitempty"`
	Kind      string    `json:"kind"`
	Payload   string    `json:"payload,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}