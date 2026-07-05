package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func readFileOrStdin(path string, stdin io.Reader) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(stdin)
	}
	return os.ReadFile(path)
}

func decodeJSON(data []byte, v any) error {
	dec := json.NewDecoder(bytesReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	return nil
}

type bytesReaderImpl struct {
	data []byte
	pos  int
}

func bytesReader(data []byte) *bytesReaderImpl {
	return &bytesReaderImpl{data: data}
}

func (r *bytesReaderImpl) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func mustBeEmpty(name string, val string) error {
	if val != "" {
		return fmt.Errorf("%s must be empty", name)
	}
	return nil
}