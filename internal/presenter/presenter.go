package presenter

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/fizza/fizza/internal/config"
	"github.com/fizza/fizza/internal/dbutil"
	"github.com/fizza/fizza/internal/model"
	"github.com/rodaine/table"
)

type ColorFn func(format string, args ...any) string

func IdentityColor(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}

func AnsiCyanBold(format string, args ...any) string {
	return "\033[1m\033[36m" + fmt.Sprintf(format, args...) + "\033[0m"
}

type Renderer struct {
	w      io.Writer
	noColor bool
}

func New(w io.Writer, noColor bool) *Renderer {
	return &Renderer{w: w, noColor: noColor}
}

func (r *Renderer) color() ColorFn {
	if r.noColor {
		return IdentityColor
	}
	return AnsiCyanBold
}

func (r *Renderer) Project(p *model.Project) error {
	return r.kv(
		[]string{"ID", "Name", "Description", "Created", "Updated"},
		[]string{
			strconv.FormatInt(p.ID, 10),
			p.Name,
			p.Description,
			formatTime(p.CreatedAt),
			formatTime(p.UpdatedAt),
		},
	)
}

func (r *Renderer) ProjectList(ps []*model.Project) error {
	if len(ps) == 0 {
		return r.empty("no projects")
	}
	rows := make([][]string, len(ps))
	for i, p := range ps {
		rows[i] = []string{
			strconv.FormatInt(p.ID, 10),
			p.Name,
			p.Description,
			formatTime(p.CreatedAt),
		}
	}
	return r.table([]string{"ID", "NAME", "DESCRIPTION", "CREATED"}, rows)
}

func (r *Renderer) Board(b *model.Board) error {
	def := "no"
	if b.IsDefault {
		def = "yes"
	}
	return r.kv(
		[]string{"ID", "Project", "Name", "Default", "Created"},
		[]string{
			strconv.FormatInt(b.ID, 10),
			strconv.FormatInt(b.ProjectID, 10),
			b.Name,
			def,
			formatTime(b.CreatedAt),
		},
	)
}

func (r *Renderer) BoardList(bs []*model.Board) error {
	if len(bs) == 0 {
		return r.empty("no boards")
	}
	rows := make([][]string, len(bs))
	for i, b := range bs {
		def := ""
		if b.IsDefault {
			def = "yes"
		}
		rows[i] = []string{
			strconv.FormatInt(b.ID, 10),
			b.Name,
			def,
			formatTime(b.CreatedAt),
		}
	}
	return r.table([]string{"ID", "NAME", "DEFAULT", "CREATED"}, rows)
}

func (r *Renderer) Column(c *model.Column) error {
	return r.kv(
		[]string{"ID", "Board", "Name", "Position", "Color"},
		[]string{
			strconv.FormatInt(c.ID, 10),
			strconv.FormatInt(c.BoardID, 10),
			c.Name,
			strconv.Itoa(c.Position),
			c.Color,
		},
	)
}

func (r *Renderer) ColumnList(cs []*model.Column) error {
	if len(cs) == 0 {
		return r.empty("no columns")
	}
	rows := make([][]string, len(cs))
	for i, c := range cs {
		rows[i] = []string{
			strconv.FormatInt(c.ID, 10),
			c.Name,
			strconv.Itoa(c.Position),
			c.Color,
		}
	}
	return r.table([]string{"ID", "NAME", "POSITION", "COLOR"}, rows)
}

func (r *Renderer) Task(t *model.Task) error {
	due := ""
	if t.DueDate != nil {
		due = t.DueDate.Format("2006-01-02")
	}
	parent := ""
	if t.ParentID != nil {
		parent = strconv.FormatInt(*t.ParentID, 10)
	}
	return r.kv(
		[]string{"ID", "Board", "Status", "Title", "Priority", "Parent", "Due", "Created", "Updated"},
		[]string{
			strconv.FormatInt(t.ID, 10),
			strconv.FormatInt(t.BoardID, 10),
			t.ColumnName,
			t.Title,
			t.Priority.String(),
			parent,
			due,
			formatTime(t.CreatedAt),
			formatTime(t.UpdatedAt),
		},
	)
}

func (r *Renderer) TaskList(ts []*model.Task) error {
	if len(ts) == 0 {
		return r.empty("no tasks")
	}
	rows := make([][]string, len(ts))
	for i, t := range ts {
		due := ""
		if t.DueDate != nil {
			due = t.DueDate.Format("2006-01-02")
		}
		rows[i] = []string{
			strconv.FormatInt(t.ID, 10),
			t.ColumnName,
			t.Title,
			t.Priority.String(),
			due,
			formatTime(t.UpdatedAt),
		}
	}
	return r.table([]string{"ID", "STATUS", "TITLE", "PRIORITY", "DUE", "UPDATED"}, rows)
}

func (r *Renderer) Config(c config.Config) error {
	project := c.Project
	if project == "" {
		project = "(unset)"
	}
	return r.kv(
		[]string{"Mode", "Project"},
		[]string{c.Mode, project},
	)
}

func (r *Renderer) KV(key, value string) error {
	return r.kv([]string{key}, []string{value})
}

func (r *Renderer) Write(p []byte) (int, error) {
	return r.w.Write(p)
}

func (r *Renderer) empty(msg string) error {
	_, err := fmt.Fprintln(r.w, msg)
	return err
}

func (r *Renderer) table(headers []string, rows [][]string) error {
	hdrArgs := make([]any, len(headers))
	for i, h := range headers {
		hdrArgs[i] = h
	}
	tbl := table.New(hdrArgs...)
	color := r.color()
	if !r.noColor {
		tbl = tbl.WithHeaderFormatter(func(format string, vals ...interface{}) string {
			return color(format, vals...)
		})
	}
	for _, row := range rows {
		args := make([]any, len(row))
		for i, v := range row {
			args[i] = v
		}
		tbl = tbl.AddRow(args...)
	}
	tbl = tbl.WithWriter(r.w)
	tbl.Print()
	return nil
}

func (r *Renderer) kv(headers, row []string) error {
	width := 0
	for _, h := range headers {
		if len(h) > width {
			width = len(h)
		}
	}
	color := r.color()
	for i, h := range headers {
		if i >= len(row) {
			break
		}
		key := padRight(h+":", width+1)
		if r.noColor {
			fmt.Fprintf(r.w, "%s  %s\n", key, row[i])
		} else {
			fmt.Fprintf(r.w, "%s  %s\n", color(key), row[i])
		}
	}
	return nil
}

func formatTime(t dbutil.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04")
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}