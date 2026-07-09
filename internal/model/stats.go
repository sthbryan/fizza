package model

type NamedCount struct {
	Name  string `json:"name" db:"name"`
	Count int64  `json:"count" db:"count"`
}

type DayCount struct {
	Date  string `json:"date" db:"date"`
	Count int64  `json:"count" db:"count"`
}

type ProjectStatsRow struct {
	Name     string `json:"name" db:"name"`
	Boards   int64  `json:"boards" db:"boards"`
	Tasks    int64  `json:"tasks" db:"tasks"`
	Done     int64  `json:"done" db:"done"`
	Open     int64  `json:"open" db:"open"`
	Overdue  int64  `json:"overdue" db:"overdue"`
	Archived int64  `json:"archived" db:"archived"`
}

type BoardStatsRow struct {
	Project  string `json:"project" db:"project"`
	Name     string `json:"name" db:"name"`
	Tasks    int64  `json:"tasks" db:"tasks"`
	Done     int64  `json:"done" db:"done"`
	Open     int64  `json:"open" db:"open"`
	Overdue  int64  `json:"overdue" db:"overdue"`
	Archived int64  `json:"archived" db:"archived"`
}

type StatsScope struct {
	Project string `json:"project,omitempty"`
	Board   string `json:"board,omitempty"`
}

type StatsTotals struct {
	Projects int64 `json:"projects"`
	Boards   int64 `json:"boards"`
	Tasks    int64 `json:"tasks"`
	Done     int64 `json:"done"`
	Open     int64 `json:"open"`
	Overdue  int64 `json:"overdue"`
	Archived int64 `json:"archived"`
}

type Stats struct {
	Scope         StatsScope        `json:"scope"`
	Totals        StatsTotals       `json:"totals"`
	ByPriority    []NamedCount      `json:"by_priority"`
	ByColumn      []NamedCount      `json:"by_column"`
	ByProject     []ProjectStatsRow `json:"by_project,omitempty"`
	ByBoard       []BoardStatsRow   `json:"by_board,omitempty"`
	CreatedByDay  []DayCount        `json:"created_by_day"`
	ActivityByDay []DayCount        `json:"activity_by_day"`
}
