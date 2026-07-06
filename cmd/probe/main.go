package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/fizza/fizza/internal/db"
	"github.com/fizza/fizza/internal/httpapi"
	"github.com/fizza/fizza/internal/model"
)

func main() {
	conn, err := db.Open(context.Background(), ":memory:")
	if err != nil {
		fmt.Println("open:", err)
		os.Exit(1)
	}
	defer conn.Close()
	ctx := context.Background()
	_, err = db.CreateProject(ctx, conn, "alpha", "")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	srv := httpapi.New(conn, "alpha")
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	post := func(path string, body string) (int, []byte) {
		req, _ := http.NewRequest("POST", ts.URL+path, bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := http.DefaultClient.Do(req)
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return resp.StatusCode, b
	}
	patch := func(path string, body string) (int, []byte) {
		req, _ := http.NewRequest("PATCH", ts.URL+path, bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := http.DefaultClient.Do(req)
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return resp.StatusCode, b
	}

	status, b := post("/v1/projects/alpha/boards/main/tasks", `{"title":"task","priority":"low"}`)
	fmt.Println("create:", status, string(b))
	var env struct {
		OK    bool            `json:"ok"`
		Data  json.RawMessage `json:"data"`
		Error json.RawMessage `json:"error"`
	}
	json.Unmarshal(b, &env)
	var task model.Task
	json.Unmarshal(env.Data, &task)
	fmt.Println("task.ID:", task.ID, "due:", task.DueDate)

	status, b = patch(fmt.Sprintf("/v1/tasks/%d", task.ID), `{"title":"renamed","priority":"urgent","due":"2031-05-06"}`)
	fmt.Println("patch1:", status, string(b))
	json.Unmarshal(b, &env)
	var updated model.Task
	json.Unmarshal(env.Data, &updated)
	fmt.Println("after patch1: title=", updated.Title, "due=", updated.DueDate)

	status, b = patch(fmt.Sprintf("/v1/tasks/%d", task.ID), `{"clear_due":true}`)
	fmt.Println("clear_due:", status, string(b))
	json.Unmarshal(b, &env)
	json.Unmarshal(env.Data, &updated)
	fmt.Println("after clear_due: title=", updated.Title, "due=", updated.DueDate, "nil?", updated.DueDate == nil)
}