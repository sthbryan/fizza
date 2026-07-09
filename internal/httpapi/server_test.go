package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fizza/fizza/internal/db"
	"github.com/fizza/fizza/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T, project string) (*httptest.Server, *Server) {
	t.Helper()
	conn, err := db.Open(context.Background(), ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	ctx := context.Background()
	if project != "" {
		_, err = db.CreateProject(ctx, conn, project, "")
		require.NoError(t, err)
	}
	s := New(conn, project)
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	return ts, s
}

func decode(t *testing.T, body io.Reader) envelope {
	t.Helper()
	var env envelope
	require.NoError(t, json.NewDecoder(body).Decode(&env))
	return env
}

func doJSON(t *testing.T, method, url string, body any) (*http.Response, []byte) {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, rdr)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	_ = resp.Body.Close()
	return resp, data
}

func TestHealth(t *testing.T) {
	ts, _ := newTestServer(t, "")
	resp, body := doJSON(t, "GET", ts.URL+"/healthz", nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	env := decode(t, bytes.NewReader(body))
	assert.True(t, env.OK)
	var data map[string]string
	require.NoError(t, json.Unmarshal(env.Data, &data))
	assert.Equal(t, "ok", data["status"])
}

func TestProjectsLifecycle(t *testing.T) {
	ts, _ := newTestServer(t, "")

	resp, body := doJSON(t, "POST", ts.URL+"/v1/projects", map[string]any{
		"name": "alpha", "description": "first",
	})
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	env := decode(t, bytes.NewReader(body))
	assert.True(t, env.OK)

	resp, body = doJSON(t, "GET", ts.URL+"/v1/projects/alpha", nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	env = decode(t, bytes.NewReader(body))
	assert.True(t, env.OK)

	resp, body = doJSON(t, "GET", ts.URL+"/v1/projects", nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	env = decode(t, bytes.NewReader(body))
	assert.True(t, env.OK)
	var list []*model.Project
	require.NoError(t, json.Unmarshal(env.Data, &list))
	assert.Len(t, list, 1)
	assert.Equal(t, "alpha", list[0].Name)

	resp, body = doJSON(t, "POST", ts.URL+"/v1/projects", map[string]any{"name": "alpha"})
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	env = decode(t, bytes.NewReader(body))
	assert.False(t, env.OK)
	require.NotNil(t, env.Error)
	assert.Equal(t, "DUPLICATE", env.Error.Code)

	resp, body = doJSON(t, "GET", ts.URL+"/v1/projects/missing", nil)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	env = decode(t, bytes.NewReader(body))
	assert.False(t, env.OK)
	require.NotNil(t, env.Error)
	assert.Equal(t, "NOT_FOUND", env.Error.Code)

	resp, body = doJSON(t, "DELETE", ts.URL+"/v1/projects/alpha", nil)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	env = decode(t, bytes.NewReader(body))
	assert.False(t, env.OK)
	assert.Equal(t, "CONFLICT", env.Error.Code)

	resp, body = doJSON(t, "DELETE", ts.URL+"/v1/projects/alpha?force=true", nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	env = decode(t, bytes.NewReader(body))
	assert.True(t, env.OK)
}

func TestCreateProject_Validation(t *testing.T) {
	ts, _ := newTestServer(t, "")
	resp, body := doJSON(t, "POST", ts.URL+"/v1/projects", map[string]any{})
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	env := decode(t, bytes.NewReader(body))
	assert.False(t, env.OK)
	require.NotNil(t, env.Error)
	assert.Equal(t, "VALIDATION", env.Error.Code)
}

func TestCreateProject_InvalidJSON(t *testing.T) {
	ts, _ := newTestServer(t, "")
	req, err := http.NewRequest("POST", ts.URL+"/v1/projects", strings.NewReader("{not-json"))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	data, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	env := decode(t, bytes.NewReader(data))
	assert.False(t, env.OK)
	assert.Equal(t, "VALIDATION", env.Error.Code)
}

func TestCreateProject_UnknownField(t *testing.T) {
	ts, _ := newTestServer(t, "")
	resp, body := doJSON(t, "POST", ts.URL+"/v1/projects", map[string]any{
		"name": "x", "bogus": "field",
	})
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	env := decode(t, bytes.NewReader(body))
	assert.False(t, env.OK)
}

func TestBoardsLifecycle(t *testing.T) {
	ts, _ := newTestServer(t, "alpha")

	resp, body := doJSON(t, "POST", ts.URL+"/v1/projects/alpha/boards", map[string]any{
		"name": "sprint-1", "columns": "todo,doing,done",
	})
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	env := decode(t, bytes.NewReader(body))
	assert.True(t, env.OK)
	var board model.Board
	require.NoError(t, json.Unmarshal(env.Data, &board))
	assert.Equal(t, "sprint-1", board.Name)

	resp, body = doJSON(t, "GET", ts.URL+"/v1/projects/alpha/boards", nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	env = decode(t, bytes.NewReader(body))
	assert.True(t, env.OK)
	var boards []*model.Board
	require.NoError(t, json.Unmarshal(env.Data, &boards))
	assert.Len(t, boards, 2)

	resp, body = doJSON(t, "GET", ts.URL+"/v1/projects/alpha/boards/sprint-1", nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	env = decode(t, bytes.NewReader(body))
	assert.True(t, env.OK)

	resp, body = doJSON(t, "GET", ts.URL+"/v1/projects/alpha/boards/nope", nil)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	env = decode(t, bytes.NewReader(body))
	assert.False(t, env.OK)
	assert.Equal(t, "NOT_FOUND", env.Error.Code)

	resp, body = doJSON(t, "POST", ts.URL+"/v1/projects/alpha/boards", map[string]any{
		"name": "sprint-1",
	})
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	env = decode(t, bytes.NewReader(body))
	assert.False(t, env.OK)
	assert.Equal(t, "DUPLICATE", env.Error.Code)

	resp, body = doJSON(t, "DELETE", ts.URL+"/v1/projects/alpha/boards/sprint-1", nil)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	env = decode(t, bytes.NewReader(body))
	assert.Equal(t, "CONFLICT", env.Error.Code)

	resp, body = doJSON(t, "DELETE", ts.URL+"/v1/projects/alpha/boards/sprint-1?force=true", nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	env = decode(t, bytes.NewReader(body))
	assert.True(t, env.OK)
}

func TestCreateBoard_NoProject(t *testing.T) {
	ts, _ := newTestServer(t, "")
	resp, body := doJSON(t, "POST", ts.URL+"/v1/projects/missing/boards", map[string]any{"name": "b"})
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	env := decode(t, bytes.NewReader(body))
	assert.False(t, env.OK)
	assert.Equal(t, "NOT_FOUND", env.Error.Code)
}

func TestTasksLifecycle(t *testing.T) {
	ts, _ := newTestServer(t, "alpha")

	resp, body := doJSON(t, "POST", ts.URL+"/v1/projects/alpha/boards/main/tasks", map[string]any{
		"title":       "first task",
		"description": "do thing",
		"priority":    "high",
		"due":         "2030-01-02",
	})
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	env := decode(t, bytes.NewReader(body))
	assert.True(t, env.OK)
	var t1 model.Task
	require.NoError(t, json.Unmarshal(env.Data, &t1))
	assert.NotZero(t, t1.ID)
	assert.Equal(t, "first task", t1.Title)
	assert.Equal(t, "high", t1.Priority.String())
	require.NotNil(t, t1.DueDate)
	assert.Equal(t, "2030-01-02", t1.DueDate.Format("2006-01-02"))

	resp, body = doJSON(t, "POST", ts.URL+"/v1/projects/alpha/boards/main/tasks", map[string]any{
		"title":  "second",
		"column": "done",
	})
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	env = decode(t, bytes.NewReader(body))
	assert.True(t, env.OK)
	var t2 model.Task
	require.NoError(t, json.Unmarshal(env.Data, &t2))
	assert.Equal(t, "done", t2.ColumnName)

	resp, body = doJSON(t, "GET", ts.URL+"/v1/projects/alpha/boards/main/tasks", nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	env = decode(t, bytes.NewReader(body))
	assert.True(t, env.OK)
	var list []*model.Task
	require.NoError(t, json.Unmarshal(env.Data, &list))
	assert.Len(t, list, 2)

	resp, body = doJSON(t, "GET", ts.URL+"/v1/projects/alpha/boards/main/tasks?column=done", nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	env = decode(t, bytes.NewReader(body))
	assert.True(t, env.OK)
	var filtered []*model.Task
	require.NoError(t, json.Unmarshal(env.Data, &filtered))
	assert.Len(t, filtered, 1)
	assert.Equal(t, t2.ID, filtered[0].ID)

	resp, body = doJSON(t, "GET", ts.URL+"/v1/projects/alpha/boards/main/tasks?priority=high,urgent", nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	env = decode(t, bytes.NewReader(body))
	assert.True(t, env.OK)
	var pri []*model.Task
	require.NoError(t, json.Unmarshal(env.Data, &pri))
	assert.Len(t, pri, 1)

	resp, body = doJSON(t, "GET", ts.URL+"/v1/projects/alpha/boards/main/tasks?search=first", nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	env = decode(t, bytes.NewReader(body))
	assert.True(t, env.OK)
	var searchRes []*model.Task
	require.NoError(t, json.Unmarshal(env.Data, &searchRes))
	assert.Len(t, searchRes, 1)

	resp, body = doJSON(t, "GET", fmt.Sprintf("%s/v1/tasks/%d", ts.URL, t1.ID), nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	env = decode(t, bytes.NewReader(body))
	assert.True(t, env.OK)
	var fetched model.Task
	require.NoError(t, json.Unmarshal(env.Data, &fetched))
	assert.Equal(t, t1.ID, fetched.ID)

	resp, body = doJSON(t, "GET", ts.URL+"/v1/tasks/999999", nil)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	env = decode(t, bytes.NewReader(body))
	assert.False(t, env.OK)
	assert.Equal(t, "NOT_FOUND", env.Error.Code)
}

func TestTasksFilter_BadPriority(t *testing.T) {
	ts, _ := newTestServer(t, "alpha")
	resp, body := doJSON(t, "GET", ts.URL+"/v1/projects/alpha/boards/main/tasks?priority=nope", nil)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	env := decode(t, bytes.NewReader(body))
	assert.False(t, env.OK)
	assert.Equal(t, "VALIDATION", env.Error.Code)
}

func TestUpdateTask(t *testing.T) {
	ts, _ := newTestServer(t, "alpha")
	resp, body := doJSON(t, "POST", ts.URL+"/v1/projects/alpha/boards/main/tasks", map[string]any{
		"title": "task", "priority": "low",
	})
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	env := decode(t, bytes.NewReader(body))
	var task model.Task
	require.NoError(t, json.Unmarshal(env.Data, &task))

	resp, body = doJSON(t, "PATCH", fmt.Sprintf("%s/v1/tasks/%d", ts.URL, task.ID), map[string]any{
		"title":    "renamed",
		"priority": "urgent",
		"due":      "2031-05-06",
	})
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	env = decode(t, bytes.NewReader(body))
	assert.True(t, env.OK)
	var updated model.Task
	require.NoError(t, json.Unmarshal(env.Data, &updated))
	assert.Equal(t, "renamed", updated.Title)
	assert.Equal(t, "urgent", updated.Priority.String())
	require.NotNil(t, updated.DueDate)

	resp, body = doJSON(t, "PATCH", fmt.Sprintf("%s/v1/tasks/%d", ts.URL, task.ID), map[string]any{
		"clear_due": true,
	})
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	env = decode(t, bytes.NewReader(body))
	assert.True(t, env.OK)
	var cleared model.Task
	require.NoError(t, json.Unmarshal(env.Data, &cleared))
	assert.Nil(t, cleared.DueDate)
	updated = cleared

	resp, body = doJSON(t, "PATCH", fmt.Sprintf("%s/v1/tasks/%d", ts.URL, task.ID), map[string]any{
		"priority": "wrong",
	})
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	env = decode(t, bytes.NewReader(body))
	assert.False(t, env.OK)
	assert.Equal(t, "VALIDATION", env.Error.Code)
}

func TestUpdateTask_ClearParent(t *testing.T) {
	ts, _ := newTestServer(t, "alpha")
	resp, body := doJSON(t, "POST", ts.URL+"/v1/projects/alpha/boards/main/tasks", map[string]any{
		"title": "parent",
	})
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	env := decode(t, bytes.NewReader(body))
	var parent model.Task
	require.NoError(t, json.Unmarshal(env.Data, &parent))

	resp, body = doJSON(t, "POST", ts.URL+"/v1/projects/alpha/boards/main/tasks", map[string]any{
		"title":  "child",
		"parent": fmt.Sprintf("%d", parent.ID),
	})
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	env = decode(t, bytes.NewReader(body))
	var child model.Task
	require.NoError(t, json.Unmarshal(env.Data, &child))
	require.NotNil(t, child.ParentID)
	assert.Equal(t, parent.ID, *child.ParentID)

	resp, body = doJSON(t, "PATCH", fmt.Sprintf("%s/v1/tasks/%d", ts.URL, child.ID), map[string]any{
		"clear_parent": true,
	})
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	env = decode(t, bytes.NewReader(body))
	assert.True(t, env.OK)
	var cleared model.Task
	require.NoError(t, json.Unmarshal(env.Data, &cleared))
	assert.Nil(t, cleared.ParentID)
}

func TestUpdateTask_CycleRejected(t *testing.T) {
	ts, _ := newTestServer(t, "alpha")
	_, body := doJSON(t, "POST", ts.URL+"/v1/projects/alpha/boards/main/tasks", map[string]any{"title": "a"})
	env := decode(t, bytes.NewReader(body))
	var a model.Task
	require.NoError(t, json.Unmarshal(env.Data, &a))

	_, body = doJSON(t, "POST", ts.URL+"/v1/projects/alpha/boards/main/tasks", map[string]any{
		"title": "b", "parent": fmt.Sprintf("%d", a.ID),
	})
	env = decode(t, bytes.NewReader(body))
	var b model.Task
	require.NoError(t, json.Unmarshal(env.Data, &b))

	resp, body := doJSON(t, "PATCH", fmt.Sprintf("%s/v1/tasks/%d", ts.URL, a.ID), map[string]any{
		"parent": fmt.Sprintf("%d", b.ID),
	})
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	env = decode(t, bytes.NewReader(body))
	assert.False(t, env.OK)
	assert.Equal(t, "VALIDATION", env.Error.Code)
}

func TestMoveTask(t *testing.T) {
	ts, _ := newTestServer(t, "alpha")
	resp, body := doJSON(t, "POST", ts.URL+"/v1/projects/alpha/boards/main/tasks", map[string]any{"title": "move me"})
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	env := decode(t, bytes.NewReader(body))
	var t1 model.Task
	require.NoError(t, json.Unmarshal(env.Data, &t1))

	resp, body = doJSON(t, "POST", fmt.Sprintf("%s/v1/tasks/%d/move", ts.URL, t1.ID), map[string]any{
		"board": "main", "column": "in_progress",
	})
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	env = decode(t, bytes.NewReader(body))
	assert.True(t, env.OK)
	var moved model.Task
	require.NoError(t, json.Unmarshal(env.Data, &moved))
	assert.Equal(t, "in_progress", moved.ColumnName)

	resp, body = doJSON(t, "POST", ts.URL+"/v1/projects/alpha/boards/main/tasks", map[string]any{"title": "z"})
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	env = decode(t, bytes.NewReader(body))
	var t2 model.Task
	require.NoError(t, json.Unmarshal(env.Data, &t2))

	resp, body = doJSON(t, "POST", fmt.Sprintf("%s/v1/tasks/%d/move", ts.URL, t2.ID), map[string]any{
		"board": "main", "column": "todo", "top": true,
	})
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	env = decode(t, bytes.NewReader(body))
	assert.True(t, env.OK)
	var topTask model.Task
	require.NoError(t, json.Unmarshal(env.Data, &topTask))
	assert.Equal(t, "todo", topTask.ColumnName)

	resp, body = doJSON(t, "POST", fmt.Sprintf("%s/v1/tasks/%d/move", ts.URL, t1.ID), map[string]any{
		"board": "main", "column": "nope",
	})
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	env = decode(t, bytes.NewReader(body))
	assert.False(t, env.OK)
	assert.Equal(t, "NOT_FOUND", env.Error.Code)
}

func TestDeleteTask(t *testing.T) {
	ts, _ := newTestServer(t, "alpha")
	resp, body := doJSON(t, "POST", ts.URL+"/v1/projects/alpha/boards/main/tasks", map[string]any{"title": "x"})
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	env := decode(t, bytes.NewReader(body))
	var t1 model.Task
	require.NoError(t, json.Unmarshal(env.Data, &t1))

	resp, body = doJSON(t, "DELETE", fmt.Sprintf("%s/v1/tasks/%d", ts.URL, t1.ID), nil)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	env = decode(t, bytes.NewReader(body))
	assert.Equal(t, "CONFLICT", env.Error.Code)

	resp, body = doJSON(t, "DELETE", fmt.Sprintf("%s/v1/tasks/%d?force=true", ts.URL, t1.ID), nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	env = decode(t, bytes.NewReader(body))
	assert.True(t, env.OK)

	resp, body = doJSON(t, "GET", fmt.Sprintf("%s/v1/tasks/%d", ts.URL, t1.ID), nil)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestDeleteTask_Unknown(t *testing.T) {
	ts, _ := newTestServer(t, "alpha")
	resp, _ := doJSON(t, "DELETE", ts.URL+"/v1/tasks/999999?force=true", nil)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestGetTask_Prefix(t *testing.T) {
	ts, _ := newTestServer(t, "alpha")
	resp, body := doJSON(t, "POST", ts.URL+"/v1/projects/alpha/boards/main/tasks", map[string]any{"title": "x"})
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	env := decode(t, bytes.NewReader(body))
	var t1 model.Task
	require.NoError(t, json.Unmarshal(env.Data, &t1))

	prefix := fmt.Sprintf("%d", t1.ID)
	if len(prefix) > 1 {
		prefix = prefix[:1]
	}
	resp, body = doJSON(t, "GET", ts.URL+"/v1/tasks/"+prefix, nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	env = decode(t, bytes.NewReader(body))
	assert.True(t, env.OK)
}

func TestCreateTask_NoBoard(t *testing.T) {
	ts, _ := newTestServer(t, "alpha")
	resp, _ := doJSON(t, "POST", ts.URL+"/v1/projects/alpha/boards/nope/tasks", map[string]any{"title": "x"})
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestRun_ContextCancel(t *testing.T) {
	conn, err := db.Open(context.Background(), ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	s := New(conn, "")
	ctx, cancel := context.WithCancel(context.Background())
	doneCh := make(chan error, 1)
	go func() {
		doneCh <- s.Run(ctx, Options{Addr: "127.0.0.1:0", ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second})
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case err := <-doneCh:
		assert.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not return after ctx cancel")
	}
}

func TestParseBool(t *testing.T) {
	for _, v := range []string{"true", "TRUE", "1", "yes", "y", "t"} {
		assert.True(t, parseBool(v), v)
	}
	for _, v := range []string{"false", "0", "", "no", "off"} {
		assert.False(t, parseBool(v), v)
	}
}

func TestWebIndex(t *testing.T) {
	ts, _ := newTestServer(t, "")
	resp, err := http.Get(ts.URL + "/")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/html")
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "fizza")
	assert.Contains(t, string(body), "id=\"app\"")
}

func TestWebSPADeepLinks(t *testing.T) {
	ts, _ := newTestServer(t, "")
	for _, path := range []string{"/projects", "/p/demo/b/main"} {
		resp, err := http.Get(ts.URL + path)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode, path)
		assert.Contains(t, resp.Header.Get("Content-Type"), "text/html", path)
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		require.NoError(t, err)
		assert.Contains(t, string(body), "id=\"app\"", path)
	}
}

func TestWebBuiltAssetsLinked(t *testing.T) {
	ts, _ := newTestServer(t, "")
	resp, err := http.Get(ts.URL + "/")
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	html := string(body)

	assert.True(t,
		strings.Contains(html, "/assets/") || strings.Contains(html, "src="),
		"index should reference frontend assets")
}

func TestWebStaticAssets(t *testing.T) {
	ts, _ := newTestServer(t, "")

	resp, err := http.Get(ts.URL + "/")
	require.NoError(t, err)
	html, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	require.NoError(t, err)

	s := string(html)
	idx := strings.Index(s, "/assets/")
	require.GreaterOrEqual(t, idx, 0, "built index should reference /assets/")
	rest := s[idx:]
	end := strings.IndexAny(rest, "\"'")
	require.Greater(t, end, 0)
	assetPath := rest[:end]

	resp, err = http.Get(ts.URL + assetPath)
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotEmpty(t, body)
}

func TestBoardSnapshot(t *testing.T) {
	ts, _ := newTestServer(t, "alpha")
	resp, body := doJSON(t, "POST", ts.URL+"/v1/projects/alpha/boards/main/tasks", map[string]any{
		"title": "from web", "priority": "high",
	})
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	_ = body

	resp, body = doJSON(t, "GET", ts.URL+"/v1/projects/alpha/boards/main/snapshot", nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	env := decode(t, bytes.NewReader(body))
	assert.True(t, env.OK)
	var snap map[string]any
	require.NoError(t, json.Unmarshal(env.Data, &snap))
	assert.Equal(t, "alpha", snap["project"])
	cols, ok := snap["columns"].([]any)
	require.True(t, ok)
	assert.NotEmpty(t, cols)
}

func TestCreateColumn(t *testing.T) {
	ts, _ := newTestServer(t, "alpha")
	resp, body := doJSON(t, "POST", ts.URL+"/v1/projects/alpha/boards/main/columns", map[string]any{
		"name": "blocked",
	})
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	env := decode(t, bytes.NewReader(body))
	assert.True(t, env.OK)
	var col map[string]any
	require.NoError(t, json.Unmarshal(env.Data, &col))
	assert.Equal(t, "blocked", col["name"])

	resp, body = doJSON(t, "GET", ts.URL+"/v1/projects/alpha/boards/main/snapshot", nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	env = decode(t, bytes.NewReader(body))
	var snap map[string]any
	require.NoError(t, json.Unmarshal(env.Data, &snap))
	cols := snap["columns"].([]any)
	assert.GreaterOrEqual(t, len(cols), 5)
}

func TestDeleteColumn(t *testing.T) {
	ts, _ := newTestServer(t, "alpha")

	resp, _ := doJSON(t, "POST", ts.URL+"/v1/projects/alpha/boards/main/columns", map[string]any{
		"name": "blocked",
	})
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	resp, body := doJSON(t, "DELETE", ts.URL+"/v1/projects/alpha/boards/main/columns/blocked", nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	env := decode(t, bytes.NewReader(body))
	assert.True(t, env.OK)

	resp, _ = doJSON(t, "POST", ts.URL+"/v1/projects/alpha/boards/main/tasks", map[string]any{
		"title": "stay", "column": "todo",
	})
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	resp, body = doJSON(t, "DELETE", ts.URL+"/v1/projects/alpha/boards/main/columns/todo", nil)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	env = decode(t, bytes.NewReader(body))
	assert.False(t, env.OK)

	resp, body = doJSON(t, "DELETE", ts.URL+"/v1/projects/alpha/boards/main/columns/todo?force=true", nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	env = decode(t, bytes.NewReader(body))
	assert.True(t, env.OK)

	resp, body = doJSON(t, "GET", ts.URL+"/v1/projects/alpha/boards/main/snapshot", nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	env = decode(t, bytes.NewReader(body))
	var snap map[string]any
	require.NoError(t, json.Unmarshal(env.Data, &snap))
	cols := snap["columns"].([]any)
	for _, c := range cols {
		m := c.(map[string]any)
		assert.NotEqual(t, "todo", m["name"])
		assert.NotEqual(t, "blocked", m["name"])
	}
}
