package toon

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testBoard struct {
	ID        int64     `json:"id"`
	ProjectID int64     `json:"project_id"`
	Name      string    `json:"name"`
	IsDefault bool      `json:"is_default"`
	CreatedAt time.Time `json:"created_at"`
}

type testTask struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Priority    string `json:"priority"`
	Description string `json:"description,omitempty"`
}

func TestEncode_SliceTabular(t *testing.T) {
	boards := []testBoard{
		{ID: 1, ProjectID: 1, Name: "main", IsDefault: true, CreatedAt: time.Date(2026, 7, 7, 3, 26, 24, 0, time.UTC)},
		{ID: 2, ProjectID: 1, Name: "dev", IsDefault: false, CreatedAt: time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC)},
	}
	out, err := Encode(map[string]any{"ok": true, "data": boards})
	require.NoError(t, err)
	assert.Contains(t, out, "data[2]{id,project_id,name,is_default,created_at}:")
	assert.Contains(t, out, "1,1,main,true,2026-07-07T03:26:24Z")
	assert.Contains(t, out, "2,1,dev,false,2026-07-06T00:00:00Z")
	assert.Contains(t, out, "ok: true")
}

func TestEncode_SingleObject(t *testing.T) {
	b := testBoard{ID: 1, ProjectID: 1, Name: "main", IsDefault: true, CreatedAt: time.Date(2026, 7, 7, 3, 26, 24, 0, time.UTC)}
	out, err := Encode(map[string]any{"ok": true, "data": b})
	require.NoError(t, err)
	assert.Contains(t, out, "data:")
	assert.Contains(t, out, "id: 1")
	assert.Contains(t, out, "name: main")
	assert.Contains(t, out, "is_default: true")
}

func TestEncode_NilData(t *testing.T) {
	out, err := Encode(map[string]any{"ok": true, "data": nil})
	require.NoError(t, err)
	assert.Contains(t, out, "data: ~")
}

func TestEncode_EmptySlice(t *testing.T) {
	out, err := Encode(map[string]any{"ok": true, "data": []testBoard{}})
	require.NoError(t, err)
	assert.Contains(t, out, "data: []")
}

func TestEncode_StringQuoting(t *testing.T) {
	out, err := Encode(map[string]any{
		"ok":   true,
		"data": map[string]string{"title": "needs,quote", "tag": "ok"},
	})
	require.NoError(t, err)
	assert.Contains(t, out, `title: "needs,quote"`)
	assert.Contains(t, out, "tag: ok")
}

func TestEncode_NumericQuoting(t *testing.T) {
	out, err := Encode(map[string]any{
		"ok":   true,
		"data": map[string]string{"version": "1.0"},
	})
	require.NoError(t, err)
	assert.Contains(t, out, "version: 1.0")
}

func TestEncode_SliceNonStruct(t *testing.T) {
	out, err := Encode(map[string]any{"ok": true, "data": []string{"a", "b", "c"}})
	require.NoError(t, err)
	assert.Contains(t, out, "data[3]:")
	assert.Contains(t, out, "- a")
	assert.Contains(t, out, "- b")
	assert.Contains(t, out, "- c")
}

func TestEncode_OMIT(t *testing.T) {
	type withOmit struct {
		Name     string `json:"name"`
		Internal string `json:"-"`
	}
	out, err := Encode(withOmit{Name: "x", Internal: "secret"})
	require.NoError(t, err)
	assert.Contains(t, out, "name: x")
	assert.NotContains(t, out, "secret")
}

func TestEncode_BoolAndNumber(t *testing.T) {
	out, err := Encode(map[string]any{
		"ok":   true,
		"data": map[string]any{"count": 42, "ratio": 1.5, "ok": true, "off": false},
	})
	require.NoError(t, err)
	assert.Contains(t, out, "count: 42")
	assert.Contains(t, out, "ratio: 1.5")
	assert.Contains(t, out, "ok: true")
	assert.Contains(t, out, "off: false")
}

func TestEncode_TaskWithDescription(t *testing.T) {
	tasks := []testTask{
		{ID: 1, Title: "fix bug", Priority: "high", Description: "details,with,commas"},
		{ID: 2, Title: "ship it", Priority: "medium"},
	}
	out, err := Encode(map[string]any{"ok": true, "data": tasks})
	require.NoError(t, err)
	assert.Contains(t, out, "data[2]{id,title,priority,description}:")
	assert.Contains(t, out, `1,fix bug,high,"details,with,commas"`)
	assert.Contains(t, out, "2,ship it,medium,")
}
