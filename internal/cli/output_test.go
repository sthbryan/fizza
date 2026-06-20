package cli

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/fizza/fizza/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvelope_OKAndFail(t *testing.T) {
	got := OK(map[string]any{"id": 1, "name": "x"})
	assert.True(t, got.OK)
	assert.Nil(t, got.Error)

	got = Fail(CodeNotFound, "missing")
	assert.False(t, got.OK)
	require.NotNil(t, got.Error)
	assert.Equal(t, CodeNotFound, got.Error.Code)
}

func TestClassifyError(t *testing.T) {
	cases := []struct {
		err      error
		wantCode ErrorCode
		wantExit int
	}{
		{nil, "", ExitOK},
		{db.ErrNotFound, CodeNotFound, ExitNotFound},
		{db.ErrDuplicate, CodeDuplicate, ExitDuplicate},
	}
	for _, c := range cases {
		env, exit := ClassifyError(c.err)
		if c.err == nil {
			assert.True(t, env.OK)
			continue
		}
		assert.Equal(t, c.wantCode, env.Error.Code)
		assert.Equal(t, c.wantExit, exit)
	}
}

func TestOutput_JSON(t *testing.T) {
	var buf bytes.Buffer
	o := NewOutput(&buf, "json", true)
	env := OK([]map[string]any{{"id": 1}})
	require.NoError(t, o.Write(env))

	var decoded Envelope
	require.NoError(t, json.Unmarshal(buf.Bytes(), &decoded))
	assert.True(t, decoded.OK)
	assert.Len(t, decoded.Data.([]any), 1)
}

func TestOutput_Pretty(t *testing.T) {
	var buf bytes.Buffer
	o := NewOutput(&buf, "pretty", true)
	require.NoError(t, o.Pretty(
		[]string{"ID", "NAME"},
		[][]string{{"1", "alpha"}, {"2", "beta"}},
	))
	out := buf.String()
	assert.Contains(t, out, "ID")
	assert.Contains(t, out, "NAME")
	assert.Contains(t, out, "alpha")
	assert.Contains(t, out, "beta")
	assert.NotEmpty(t, out)
}

func TestParseFlagInt64(t *testing.T) {
	v, err := ParseFlagInt64("42")
	require.NoError(t, err)
	assert.Equal(t, int64(42), v)

	_, err = ParseFlagInt64("")
	require.Error(t, err)

	_, err = ParseFlagInt64("abc")
	require.Error(t, err)
}

func TestParseInt64(t *testing.T) {
	cases := map[string]int64{
		"0":    0,
		"1":    1,
		"42":   42,
		"9999": 9999,
	}
	for in, want := range cases {
		got, err := parseInt64(in)
		require.NoError(t, err, in)
		assert.Equal(t, want, got)
	}
	for _, bad := range []string{"", "-1", "1a", "1.0"} {
		_, err := parseInt64(bad)
		assert.Error(t, err, bad)
	}
}