package db

import "github.com/fizza/fizza/internal/model"

const terminalColumnSQL = `lower(c.name) IN ('done', 'completed', 'closed')`

func IsTerminalColumn(name string) bool {
	return model.IsTerminalColumn(name)
}
