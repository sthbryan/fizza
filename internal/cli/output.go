package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/fizza/fizza/internal/db"
	"github.com/fizza/fizza/internal/model"
	"github.com/fizza/fizza/internal/presenter"
	"github.com/fizza/fizza/internal/service"
	"github.com/fizza/fizza/internal/toon"
)

const (
	ExitOK         = 0
	ExitGeneric    = 1
	ExitNotFound   = 2
	ExitValidation = 3
	ExitDuplicate  = 4
	ExitConflict   = 5
)

const (
	FormatJSON   = "json"
	FormatPretty = "pretty"
	FormatTOON   = "toon"
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

type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *ExitError) Unwrap() error { return e.Err }

func newExitError(code int, err error) *ExitError {
	return &ExitError{Code: code, Err: err}
}

func OK(data any) Envelope { return Envelope{OK: true, Data: data} }

func Fail(code ErrorCode, msg string) Envelope {
	return Envelope{OK: false, Error: &ErrorPayload{Code: code, Message: msg}}
}

func ClassifyError(err error) (Envelope, int) {
	var ee *ExitError
	if errors.As(err, &ee) {
		switch ee.Code {
		case ExitConflict:
			return Fail(CodeConflict, conflictMessage(ee)), ExitConflict
		case ExitNotFound:
			return Fail(CodeNotFound, notFoundMessage(ee)), ExitNotFound
		case ExitDuplicate:
			return Fail(CodeDuplicate, ee.Error()), ExitDuplicate
		case ExitValidation:
			return Fail(CodeValidation, validationMessage(ee.Err)), ExitValidation
		case ExitOK:
			return OK(nil), ExitOK
		default:
			if ee.Err == nil {
				return Fail(CodeInternal, "error"), ExitGeneric
			}
			return ClassifyError(ee.Err)
		}
	}
	switch {
	case err == nil:
		return OK(nil), ExitOK
	case errors.Is(err, ErrValidation) || errors.Is(err, model.ErrValidation) || errors.Is(err, service.ErrValidation):
		return Fail(CodeValidation, validationMessage(err)), ExitValidation
	case db.IsNotFound(err):
		return Fail(CodeNotFound, err.Error()), ExitNotFound
	case db.IsDuplicate(err):
		return Fail(CodeDuplicate, err.Error()), ExitDuplicate
	case errors.Is(err, db.ErrWIPLimitReached):
		return Fail(CodeConflict, err.Error()), ExitConflict
	default:
		return Fail(CodeInternal, err.Error()), ExitGeneric
	}
}

func UserMessage(err error) string {
	if err == nil {
		return ""
	}
	env, _ := ClassifyError(err)
	if env.Error == nil {
		return err.Error()
	}
	return env.Error.Message
}

func validationMessage(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	const prefix = "validation: "
	if strings.HasPrefix(msg, prefix) {
		return msg[len(prefix):]
	}
	return msg
}

func conflictMessage(ee *ExitError) string {
	if ee.Err == nil {
		return "conflict"
	}
	return ee.Err.Error()
}

func notFoundMessage(ee *ExitError) string {
	if ee.Err == nil {
		return "not found"
	}
	return ee.Err.Error()
}

type Output struct {
	w        io.Writer
	format   string
	noColor  bool
	withMeta bool
}

func NewOutput(w io.Writer, format string, noColor bool) *Output {
	return &Output{w: w, format: format, noColor: noColor}
}

func (o *Output) WithMeta(enabled bool) *Output {
	o.withMeta = enabled
	return o
}

func (o *Output) Write(env Envelope) error {
	if !env.OK {
		return o.writeJSON(env)
	}
	switch o.format {
	case FormatPretty:
		if err := o.writePretty(env.Data); err == nil {
			return nil
		}
	case FormatTOON:
		return o.writeTOON(env)
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

func (o *Output) writeTOON(env Envelope) error {
	var out string
	var err error
	if o.withMeta {
		out, err = toon.Encode(env)
	} else {
		out, err = toon.EncodeLLM(env)
	}
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(o.w, out)
	return err
}

func (o *Output) writePretty(data any) error {
	r := presenter.New(o.w, o.noColor)
	return renderPretty(r, data)
}

func renderPretty(r *presenter.Renderer, data any) error {
	switch v := data.(type) {
	case *model.Project:
		return r.Project(v)
	case []*model.Project:
		return r.ProjectList(v)
	case *model.Board:
		return r.Board(v)
	case []*model.Board:
		return r.BoardList(v)
	case *model.Column:
		return r.Column(v)
	case []*model.Column:
		return r.ColumnList(v)
	case *model.Task:
		return r.Task(v)
	case []*model.Task:
		return r.TaskList(v)
	case BoardView:
		return renderBoardView(r, v)
	case *BoardView:
		return renderBoardView(r, *v)
	case map[string]any:
		return renderMap(r, v)
	case nil:
		_, err := fmt.Fprintln(r, "(no data)")
		return err
	default:
		return fmt.Errorf("no pretty renderer for %s", reflect.TypeOf(data))
	}
}

type BoardView struct {
	Board   *model.Board        `json:"board"`
	Columns []BoardColumnBucket `json:"columns"`
}

type BoardColumnBucket struct {
	Column *model.Column `json:"column"`
	Tasks  []*model.Task `json:"tasks"`
}

func renderBoardView(r *presenter.Renderer, v BoardView) error {
	if v.Board != nil {
		if err := r.Board(v.Board); err != nil {
			return err
		}
	}
	for _, bucket := range v.Columns {
		if bucket.Column == nil {
			continue
		}
		if _, err := fmt.Fprintf(r, "\n[ %s ]\n", bucket.Column.Name); err != nil {
			return err
		}
		if len(bucket.Tasks) == 0 {
			if _, err := fmt.Fprintln(r, "  (empty)"); err != nil {
				return err
			}
			continue
		}
		if err := r.TaskList(bucket.Tasks); err != nil {
			return err
		}
	}
	return nil
}

func renderMap(r *presenter.Renderer, m map[string]any) error {
	for k, v := range m {
		if _, err := fmt.Fprintf(r, "%s: %v\n", k, formatMapValue(v)); err != nil {
			return err
		}
	}
	return nil
}

func formatMapValue(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case time.Time:
		return t.Format("2006-01-02 15:04")
	case int64:
		return strconv.FormatInt(t, 10)
	case int:
		return strconv.Itoa(t)
	default:
		return fmt.Sprintf("%v", v)
	}
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
