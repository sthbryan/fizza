package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fizza/fizza/internal/db"
	"github.com/fizza/fizza/internal/model"
	"github.com/rodaine/table"
)

const (
	ExitOK         = 0
	ExitGeneric    = 1
	ExitNotFound   = 2
	ExitValidation = 3
	ExitDuplicate  = 4
	ExitConflict   = 5
)

type ErrorCode string

const (
	CodeNotFound   ErrorCode = "NOT_FOUND"
	CodeDuplicate  ErrorCode = "DUPLICATE"
	CodeValidation ErrorCode = "VALIDATION"
	CodeConflict   ErrorCode = "CONFLICT"
	CodeInternal   ErrorCode = "INTERNAL"
)

type ErrorPayload struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

type Envelope struct {
	OK    bool          `json:"ok"`
	Data  any           `json:"data,omitempty"`
	Error *ErrorPayload `json:"error,omitempty"`
}

func OK(data any) Envelope { return Envelope{OK: true, Data: data} }

func Fail(code ErrorCode, msg string) Envelope {
	return Envelope{OK: false, Error: &ErrorPayload{Code: code, Message: msg}}
}

func ClassifyError(err error) (Envelope, int) {
	switch {
	case err == nil:
		return OK(nil), ExitOK
	case errors.Is(err, ErrValidation):
		return Fail(CodeValidation, validationMessage(err)), ExitValidation
	case db.IsNotFound(err):
		return Fail(CodeNotFound, err.Error()), ExitNotFound
	case db.IsDuplicate(err):
		return Fail(CodeDuplicate, err.Error()), ExitDuplicate
	default:
		return Fail(CodeInternal, err.Error()), ExitGeneric
	}
}

func validationMessage(err error) string {
	msg := err.Error()
	const prefix = "validation: "
	if strings.HasPrefix(msg, prefix) {
		return msg[len(prefix):]
	}
	return msg
}

type Output struct {
	w       io.Writer
	format  string
	noColor bool
}

func NewOutput(w io.Writer, format string, noColor bool) *Output {
	return &Output{w: w, format: format, noColor: noColor}
}

func (o *Output) Write(env Envelope) error {
	if !env.OK {
		return o.writeJSON(env)
	}
	if o.format == "pretty" {
		if err := o.writePretty(env.Data); err == nil {
			return nil
		}
	}
	return o.writeJSON(env)
}

func (o *Output) writeJSON(env Envelope) error {
	b, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(o.w, string(b))
	return err
}

func (o *Output) writePretty(data any) error {
	rows, headers, single, ok := collectTable(data)
	if !ok {
		return fmt.Errorf("no pretty renderer for %T", data)
	}
	if single {
		return o.prettyKeyValue(headers, rows[0])
	}
	return o.prettyTable(headers, rows)
}

func (o *Output) prettyTable(headers []string, rows [][]string) error {
	hdrArgs := make([]any, len(headers))
	for i, h := range headers {
		hdrArgs[i] = h
	}
	tbl := table.New(hdrArgs...)
	if !o.noColor {
		tbl = tbl.WithHeaderFormatter(func(format string, vals ...interface{}) string {
			return "\033[1m\033[36m" + fmt.Sprintf(format, vals...) + "\033[0m"
		})
	}
	for _, r := range rows {
		row := make([]any, len(r))
		for i, v := range r {
			row[i] = v
		}
		tbl = tbl.AddRow(row...)
	}
	tbl = tbl.WithWriter(o.w)
	tbl.Print()
	return nil
}

func (o *Output) prettyKeyValue(headers, row []string) error {
	width := 0
	for _, h := range headers {
		if len(h) > width {
			width = len(h)
		}
	}
	for i, h := range headers {
		if i >= len(row) {
			break
		}
		key := padRight(h+":", width+1)
		if o.noColor {
			fmt.Fprintf(o.w, "%s  %s\n", key, row[i])
		} else {
			fmt.Fprintf(o.w, "\033[1m\033[36m%s\033[0m  %s\n", key, row[i])
		}
	}
	return nil
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}

func collectTable(data any) (rows [][]string, headers []string, single, ok bool) {
	switch v := data.(type) {
	case *model.Project:
		return projectRows([]*model.Project{v}), projectHeaders(), true, true
	case []*model.Project:
		if len(v) == 0 {
			return [][]string{}, projectHeaders(), false, true
		}
		return projectRows(v), projectHeaders(), false, true

	case *model.Board:
		return boardRows([]*model.Board{v}), boardHeaders(), true, true
	case []*model.Board:
		if len(v) == 0 {
			return [][]string{}, boardHeaders(), false, true
		}
		return boardRows(v), boardHeaders(), false, true

	case *model.Task:
		return taskRows([]*model.Task{v}), taskHeaders(), true, true
	case []*model.Task:
		if len(v) == 0 {
			return [][]string{}, taskHeaders(), false, true
		}
		return taskRows(v), taskHeaders(), false, true

	case *model.Column:
		return columnRows([]*model.Column{v}), columnHeaders(), true, true
	case []*model.Column:
		if len(v) == 0 {
			return [][]string{}, columnHeaders(), false, true
		}
		return columnRows(v), columnHeaders(), false, true
	}
	return nil, nil, false, false
}

func projectHeaders() []string  { return []string{"ID", "NAME", "DESCRIPTION", "CREATED"} }
func boardHeaders() []string    { return []string{"ID", "NAME", "DEFAULT", "CREATED"} }
func columnHeaders() []string  { return []string{"ID", "NAME", "POSITION", "COLOR"} }
func taskHeaders() []string    { return []string{"ID", "STATUS", "TITLE", "PRIORITY", "DUE", "UPDATED"} }

func projectRows(ps []*model.Project) [][]string {
	out := make([][]string, len(ps))
	for i, p := range ps {
		out[i] = []string{
			strconv.FormatInt(p.ID, 10),
			p.Name,
			p.Description,
			formatTime(p.CreatedAt),
		}
	}
	return out
}

func boardRows(bs []*model.Board) [][]string {
	out := make([][]string, len(bs))
	for i, b := range bs {
		defaultMark := ""
		if b.IsDefault {
			defaultMark = "yes"
		}
		out[i] = []string{
			strconv.FormatInt(b.ID, 10),
			b.Name,
			defaultMark,
			formatTime(b.CreatedAt),
		}
	}
	return out
}

func columnRows(cs []*model.Column) [][]string {
	out := make([][]string, len(cs))
	for i, c := range cs {
		out[i] = []string{
			strconv.FormatInt(c.ID, 10),
			c.Name,
			strconv.Itoa(c.Position),
			c.Color,
		}
	}
	return out
}

func taskRows(ts []*model.Task) [][]string {
	out := make([][]string, len(ts))
	for i, t := range ts {
		due := ""
		if t.DueDate != nil {
			due = t.DueDate.Format("2006-01-02")
		}
		out[i] = []string{
			strconv.FormatInt(t.ID, 10),
			t.ColumnName,
			t.Title,
			t.Priority,
			due,
			formatTime(t.UpdatedAt),
		}
	}
	return out
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04")
}

func ParseFlagInt64(s string) (int64, error) {
	if s == "" {
		return 0, errors.New("empty value")
	}
	return strconv.ParseInt(s, 10, 64)
}

func StdoutIsTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
