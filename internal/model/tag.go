package model

import (
	"strings"

	"github.com/fizza/fizza/internal/dbutil"
)

var (
	ErrTagNameEmpty = ValidationError("tag name cannot be empty")
	ErrTagNameLong  = ValidationError("tag name too long (max 64 chars)")
)

func ValidateTag(name string) error {
	if strings.TrimSpace(name) == "" {
		return ErrTagNameEmpty
	}
	if len(name) > 64 {
		return ErrTagNameLong
	}
	return nil
}

type Tag struct {
	ID        int64       `json:"id" db:"id"`
	ProjectID int64       `json:"project_id" db:"project_id"`
	Name      string      `json:"name" db:"name"`
	CreatedAt dbutil.Time `json:"created_at" db:"created_at"`
}
