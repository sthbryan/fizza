package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/fizza/fizza/internal/db"
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
	b, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(o.w, string(b))
	return err
}

func (o *Output) Pretty(headers []string, rows [][]string) error {
	hdrArgs := make([]any, len(headers))
	for i, h := range headers {
		hdrArgs[i] = h
	}
	tbl := table.New(hdrArgs...)
	if !o.noColor {
		style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
		tbl = tbl.WithHeaderFormatter(func(s string, _ ...any) string {
			return style.Render(s)
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