package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fizza/fizza/internal/db"
	"github.com/fizza/fizza/internal/model"
	"github.com/fizza/fizza/internal/service"
	"github.com/jmoiron/sqlx"
)

type Server struct {
	svc  *service.Service
	mux  *http.ServeMux
	addr string
}

func New(conn *sqlx.DB, project string) *Server {
	svc := service.New(conn, project, "", "")
	s := &Server{
		svc: svc,
		mux: http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return localGuard(s.mux)
}

func (s *Server) Service() *service.Service {
	return s.svc
}

func (s *Server) Addr() string {
	return s.addr
}

type Options struct {
	Addr         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

func (s *Server) Run(ctx context.Context, opts Options) error {
	srv := &http.Server{
		Addr:         opts.Addr,
		Handler:      s.Handler(),
		ReadTimeout:  opts.ReadTimeout,
		WriteTimeout: opts.WriteTimeout,
	}
	s.addr = opts.Addr
	errCh := make(chan error, 1)
	go func() {
		err := srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error *errorPayload   `json:"error,omitempty"`
}

type errorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	b, err := json.Marshal(payload)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(envelope{
			OK:    false,
			Error: &errorPayload{Code: "INTERNAL", Message: "marshal: " + err.Error()},
		})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(b)
}

func writeOK(w http.ResponseWriter, status int, data any) {
	raw, err := json.Marshal(data)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK:    false,
			Error: &errorPayload{Code: "INTERNAL", Message: "marshal: " + err.Error()},
		})
		return
	}
	writeJSON(w, status, envelope{OK: true, Data: raw})
}

func writeErr(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, envelope{
		OK:    false,
		Error: &errorPayload{Code: code, Message: message},
	})
}

func isJSONUnknownFieldErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "unknown field")
}

func mapError(err error) (int, string) {
	switch {
	case err == nil:
		return http.StatusOK, ""
	case errors.Is(err, model.ErrValidation):
		return http.StatusBadRequest, "VALIDATION:" + strings.TrimPrefix(err.Error(), "validation: ")
	case db.IsNotFound(err):
		return http.StatusNotFound, "NOT_FOUND:" + err.Error()
	case db.IsDuplicate(err):
		return http.StatusConflict, "DUPLICATE:" + err.Error()
	case errors.Is(err, db.ErrWIPLimitReached),
		errors.Is(err, db.ErrColumnNotEmpty),
		errors.Is(err, db.ErrLastColumn):
		return http.StatusConflict, "CONFLICT:" + err.Error()
	}
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		return http.StatusBadRequest, "VALIDATION:invalid JSON: " + err.Error()
	}
	var synErr *json.SyntaxError
	if errors.As(err, &synErr) {
		return http.StatusBadRequest, "VALIDATION:invalid JSON: " + err.Error()
	}
	if _, ok := err.(*json.InvalidUnmarshalError); ok {
		return http.StatusInternalServerError, "INTERNAL:" + err.Error()
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return http.StatusBadRequest, "VALIDATION:empty request body"
	}
	if isJSONUnknownFieldErr(err) {
		return http.StatusBadRequest, "VALIDATION:" + err.Error()
	}
	return http.StatusInternalServerError, "INTERNAL:" + err.Error()
}

func respondError(w http.ResponseWriter, err error) {
	status, codeMsg := mapError(err)
	parts := strings.SplitN(codeMsg, ":", 2)
	code := parts[0]
	msg := ""
	if len(parts) > 1 {
		msg = parts[1]
	}
	if msg == "" && err != nil {
		msg = err.Error()
	}
	writeErr(w, status, code, msg)
}

func decodeJSONBody(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	return nil
}

func (s *Server) routes() {
	s.mountWeb()

	s.mux.HandleFunc("GET /healthz", s.handleHealth)
	s.mux.HandleFunc("GET /v1/events", s.handleEventsSSE)
	s.mux.HandleFunc("GET /v1/stats", s.handleStats)

	s.mux.HandleFunc("GET /v1/projects", s.handleListProjects)
	s.mux.HandleFunc("POST /v1/projects", s.handleCreateProject)
	s.mux.HandleFunc("GET /v1/projects/{name}", s.handleGetProject)
	s.mux.HandleFunc("PATCH /v1/projects/{name}", s.handleUpdateProject)
	s.mux.HandleFunc("DELETE /v1/projects/{name}", s.handleDeleteProject)

	s.mux.HandleFunc("GET /v1/projects/{name}/boards", s.handleListBoards)
	s.mux.HandleFunc("POST /v1/projects/{name}/boards", s.handleCreateBoard)
	s.mux.HandleFunc("GET /v1/projects/{name}/boards/{board}", s.handleGetBoard)
	s.mux.HandleFunc("GET /v1/projects/{name}/boards/{board}/snapshot", s.handleBoardSnapshot)
	s.mux.HandleFunc("POST /v1/projects/{name}/boards/{board}/columns", s.handleCreateColumn)
	s.mux.HandleFunc("DELETE /v1/projects/{name}/boards/{board}/columns/{column}", s.handleDeleteColumn)
	s.mux.HandleFunc("DELETE /v1/projects/{name}/boards/{board}", s.handleDeleteBoard)
	s.mux.HandleFunc("GET /v1/projects/{name}/boards/{board}/archived", s.handleListArchived)
	s.mux.HandleFunc("POST /v1/projects/{name}/boards/{board}/archive-done", s.handleArchiveDone)

	s.mux.HandleFunc("GET /v1/projects/{name}/boards/{board}/tasks", s.handleListTasks)
	s.mux.HandleFunc("POST /v1/projects/{name}/boards/{board}/tasks", s.handleCreateTask)

	s.mux.HandleFunc("GET /v1/tasks/{id}", s.handleGetTask)
	s.mux.HandleFunc("PATCH /v1/tasks/{id}", s.handleUpdateTask)
	s.mux.HandleFunc("POST /v1/tasks/{id}/move", s.handleMoveTask)
	s.mux.HandleFunc("POST /v1/tasks/{id}/archive", s.handleArchiveTask)
	s.mux.HandleFunc("POST /v1/tasks/{id}/unarchive", s.handleUnarchiveTask)
	s.mux.HandleFunc("DELETE /v1/tasks/{id}", s.handleDeleteTask)
}

func (s *Server) handleBoardSnapshot(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectName := r.PathValue("name")
	boardName := r.PathValue("board")
	svc := service.New(s.svc.DB(), projectName, boardName, "")
	opts := service.SnapshotOpts{
		IncludeDone: parseBool(r.URL.Query().Get("include_done")),
	}
	snap, err := svc.BoardSnapshotOpts(ctx, opts)
	if err != nil {
		respondError(w, err)
		return
	}
	writeOK(w, http.StatusOK, snap)
}

func (s *Server) handleListArchived(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectName := r.PathValue("name")
	boardName := r.PathValue("board")
	svc := service.New(s.svc.DB(), projectName, boardName, "")
	tasks, err := svc.ListArchived(ctx)
	if err != nil {
		respondError(w, err)
		return
	}
	if tasks == nil {
		tasks = []*model.Task{}
	}
	writeOK(w, http.StatusOK, tasks)
}

func (s *Server) handleArchiveDone(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectName := r.PathValue("name")
	boardName := r.PathValue("board")
	svc := service.New(s.svc.DB(), projectName, boardName, "")
	n, err := svc.ArchiveDone(ctx)
	if err != nil {
		respondError(w, err)
		return
	}
	writeOK(w, http.StatusOK, map[string]any{"archived": n})
}

func (s *Server) handleArchiveTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")
	t, err := s.svc.GetTaskByPrefix(ctx, id)
	if err != nil {
		respondError(w, err)
		return
	}
	if err := s.svc.ArchiveTask(ctx, t.ID); err != nil {
		respondError(w, err)
		return
	}
	fresh, err := s.svc.GetTask(ctx, t.ID)
	if err != nil {
		respondError(w, err)
		return
	}
	writeOK(w, http.StatusOK, fresh)
}

func (s *Server) handleUnarchiveTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")
	t, err := s.svc.GetTaskByPrefix(ctx, id)
	if err != nil {
		respondError(w, err)
		return
	}
	if err := s.svc.UnarchiveTask(ctx, t.ID); err != nil {
		respondError(w, err)
		return
	}
	fresh, err := s.svc.GetTask(ctx, t.ID)
	if err != nil {
		respondError(w, err)
		return
	}
	writeOK(w, http.StatusOK, fresh)
}

type createColumnReq struct {
	Name string `json:"name"`
}

func (s *Server) handleCreateColumn(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectName := r.PathValue("name")
	boardName := r.PathValue("board")
	var in createColumnReq
	if err := decodeJSONBody(r, &in); err != nil {
		respondError(w, err)
		return
	}
	board, _, err := findBoard(ctx, s.svc.DB(), projectName, boardName)
	if err != nil {
		respondError(w, err)
		return
	}
	col, err := db.CreateColumn(ctx, s.svc.DB(), board.ID, in.Name)
	if err != nil {
		respondError(w, err)
		return
	}
	writeOK(w, http.StatusCreated, col)
}

func (s *Server) handleDeleteColumn(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectName := r.PathValue("name")
	boardName := r.PathValue("board")
	columnName := r.PathValue("column")
	force := parseBool(r.URL.Query().Get("force"))
	board, _, err := findBoard(ctx, s.svc.DB(), projectName, boardName)
	if err != nil {
		respondError(w, err)
		return
	}
	if err := db.DeleteColumn(ctx, s.svc.DB(), board.ID, columnName, force); err != nil {
		respondError(w, err)
		return
	}
	writeOK(w, http.StatusOK, map[string]any{"deleted": columnName})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeOK(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projects, err := db.ListProjects(ctx, s.svc.DB())
	if err != nil {
		respondError(w, err)
		return
	}
	if projects == nil {
		projects = []*model.Project{}
	}
	writeOK(w, http.StatusOK, projects)
}

type createProjectReq struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var in createProjectReq
	if err := decodeJSONBody(r, &in); err != nil {
		respondError(w, err)
		return
	}
	p, err := db.CreateProject(ctx, s.svc.DB(), in.Name, in.Description)
	if err != nil {
		respondError(w, err)
		return
	}
	writeOK(w, http.StatusCreated, p)
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := r.PathValue("name")
	p, err := db.GetProjectByName(ctx, s.svc.DB(), name)
	if err != nil {
		respondError(w, err)
		return
	}
	writeOK(w, http.StatusOK, p)
}

type updateProjectReq struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

func (s *Server) handleUpdateProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := r.PathValue("name")
	var in updateProjectReq
	if err := decodeJSONBody(r, &in); err != nil {
		respondError(w, err)
		return
	}
	if in.Name == nil && in.Description == nil {
		writeErr(w, http.StatusBadRequest, "VALIDATION", "provide name and/or description")
		return
	}
	p, err := db.GetProjectByName(ctx, s.svc.DB(), name)
	if err != nil {
		respondError(w, err)
		return
	}
	newName := p.Name
	newDesc := p.Description
	if in.Name != nil {
		newName = *in.Name
	}
	if in.Description != nil {
		newDesc = *in.Description
	}
	updated, err := db.UpdateProject(ctx, s.svc.DB(), p.ID, newName, newDesc)
	if err != nil {
		respondError(w, err)
		return
	}
	writeOK(w, http.StatusOK, updated)
}

func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := r.PathValue("name")
	if !parseBool(r.URL.Query().Get("force")) {
		writeErr(w, http.StatusConflict, "CONFLICT", fmt.Sprintf("refusing to delete %q without ?force=true", name))
		return
	}
	p, err := db.GetProjectByName(ctx, s.svc.DB(), name)
	if err != nil {
		respondError(w, err)
		return
	}
	if err := db.DeleteProject(ctx, s.svc.DB(), p.ID); err != nil {
		respondError(w, err)
		return
	}
	writeOK(w, http.StatusOK, map[string]any{"deleted": name, "id": p.ID})
}

func (s *Server) handleListBoards(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := r.PathValue("name")
	p, err := db.GetProjectByName(ctx, s.svc.DB(), name)
	if err != nil {
		respondError(w, err)
		return
	}
	boards, err := db.ListBoards(ctx, s.svc.DB(), p.ID)
	if err != nil {
		respondError(w, err)
		return
	}
	if boards == nil {
		boards = []*model.Board{}
	}
	writeOK(w, http.StatusOK, boards)
}

type createBoardReq struct {
	Name    string `json:"name"`
	Columns string `json:"columns,omitempty"`
}

func (s *Server) handleCreateBoard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := r.PathValue("name")
	var in createBoardReq
	if err := decodeJSONBody(r, &in); err != nil {
		respondError(w, err)
		return
	}
	p, err := db.GetProjectByName(ctx, s.svc.DB(), name)
	if err != nil {
		respondError(w, err)
		return
	}
	cols := service.SplitColumns(in.Columns)
	b, err := db.CreateBoardWithColumns(ctx, s.svc.DB(), p.ID, in.Name, cols)
	if err != nil {
		respondError(w, err)
		return
	}
	writeOK(w, http.StatusCreated, b)
}

func (s *Server) handleGetBoard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectName := r.PathValue("name")
	boardName := r.PathValue("board")
	board, _, err := findBoard(ctx, s.svc.DB(), projectName, boardName)
	if err != nil {
		respondError(w, err)
		return
	}
	cols, err := db.ListColumns(ctx, s.svc.DB(), board.ID)
	if err != nil {
		respondError(w, err)
		return
	}
	if cols == nil {
		cols = []*model.Column{}
	}
	writeOK(w, http.StatusOK, map[string]any{
		"board":   board,
		"columns": cols,
	})
}

func (s *Server) handleDeleteBoard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectName := r.PathValue("name")
	boardName := r.PathValue("board")
	if !parseBool(r.URL.Query().Get("force")) {
		writeErr(w, http.StatusConflict, "CONFLICT", fmt.Sprintf("refusing to delete board %q without ?force=true", boardName))
		return
	}
	board, _, err := findBoard(ctx, s.svc.DB(), projectName, boardName)
	if err != nil {
		respondError(w, err)
		return
	}
	if err := db.DeleteBoard(ctx, s.svc.DB(), board.ID); err != nil {
		respondError(w, err)
		return
	}
	writeOK(w, http.StatusOK, map[string]any{"deleted": board.Name, "id": board.ID})
}

func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectName := r.PathValue("name")
	boardName := r.PathValue("board")
	board, _, err := findBoard(ctx, s.svc.DB(), projectName, boardName)
	if err != nil {
		respondError(w, err)
		return
	}
	filter, err := buildTaskFilter(r.URL.Query())
	if err != nil {
		writeErr(w, http.StatusBadRequest, "VALIDATION", err.Error())
		return
	}
	tasks, err := db.ListTasksInBoard(ctx, s.svc.DB(), board.ID, filter)
	if err != nil {
		respondError(w, err)
		return
	}
	if tasks == nil {
		tasks = []*model.Task{}
	}
	writeOK(w, http.StatusOK, tasks)
}

type createTaskReq struct {
	Title       string `json:"title"`
	Column      string `json:"column,omitempty"`
	Description string `json:"description,omitempty"`
	Priority    string `json:"priority,omitempty"`
	Due         string `json:"due,omitempty"`
	Parent      string `json:"parent,omitempty"`
}

func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectName := r.PathValue("name")
	boardName := r.PathValue("board")
	var in createTaskReq
	if err := decodeJSONBody(r, &in); err != nil {
		respondError(w, err)
		return
	}
	board, _, err := findBoard(ctx, s.svc.DB(), projectName, boardName)
	if err != nil {
		respondError(w, err)
		return
	}

	pri, err := model.NewPriority(in.Priority)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "VALIDATION", err.Error())
		return
	}

	input := service.TaskCreateInput{
		Title:       in.Title,
		Description: in.Description,
		Priority:    pri,
	}
	if in.Due != "" {
		due, err := service.ParseDue(in.Due)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "VALIDATION", "due: "+err.Error())
			return
		}
		input.DueDate = due
	}
	if in.Parent != "" {
		pid, err := s.svc.GetTaskByPrefix(ctx, in.Parent)
		if err != nil {
			respondError(w, err)
			return
		}
		input.ParentID = &pid.ID
	}
	if in.Column != "" {
		col, err := db.GetColumnByName(ctx, s.svc.DB(), board.ID, in.Column)
		if err != nil {
			respondError(w, err)
			return
		}
		input.ColumnID = col.ID
	}

	svcForBoard := service.New(s.svc.DB(), projectName, boardName, in.Column)
	t, err := svcForBoard.CreateTask(ctx, input)
	if err != nil {
		respondError(w, err)
		return
	}
	writeOK(w, http.StatusCreated, t)
}

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")
	t, err := s.svc.GetTaskByPrefix(ctx, id)
	if err != nil {
		respondError(w, err)
		return
	}
	writeOK(w, http.StatusOK, t)
}

type updateTaskReq struct {
	Title       *string `json:"title,omitempty"`
	Desc        *string `json:"desc,omitempty"`
	Priority    *string `json:"priority,omitempty"`
	Due         *string `json:"due,omitempty"`
	ClearDue    bool    `json:"clear_due,omitempty"`
	Parent      *string `json:"parent,omitempty"`
	ClearParent bool    `json:"clear_parent,omitempty"`
}

func (s *Server) handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")
	var in updateTaskReq
	if err := decodeJSONBody(r, &in); err != nil {
		respondError(w, err)
		return
	}
	t, err := s.svc.GetTaskByPrefix(ctx, id)
	if err != nil {
		respondError(w, err)
		return
	}
	patch, err := buildTaskPatch(ctx, in, s.svc)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "VALIDATION", err.Error())
		return
	}
	updated, err := s.svc.UpdateTask(ctx, t.ID, patch)
	if err != nil {
		respondError(w, err)
		return
	}
	writeOK(w, http.StatusOK, updated)
}

type moveTaskReq struct {
	Project string `json:"project,omitempty"`
	Board   string `json:"board"`
	Column  string `json:"column"`
	Before  string `json:"before,omitempty"`
	After   string `json:"after,omitempty"`
	Top     bool   `json:"top,omitempty"`
}

func (s *Server) handleMoveTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")
	var in moveTaskReq
	if err := decodeJSONBody(r, &in); err != nil {
		respondError(w, err)
		return
	}
	t, err := s.svc.GetTaskByPrefix(ctx, id)
	if err != nil {
		respondError(w, err)
		return
	}
	var board *model.Board
	if in.Project != "" {
		b, _, err := findBoard(ctx, s.svc.DB(), in.Project, in.Board)
		if err != nil {
			respondError(w, err)
			return
		}
		board = b
	} else {
		b, _, err := findBoardByName(ctx, s.svc.DB(), in.Board)
		if err != nil {
			respondError(w, err)
			return
		}
		board = b
	}
	if t.BoardID != board.ID {
		writeErr(w, http.StatusBadRequest, "VALIDATION", "task belongs to a different board")
		return
	}
	col, err := db.GetColumnByName(ctx, s.svc.DB(), board.ID, in.Column)
	if err != nil {
		respondError(w, err)
		return
	}
	beforeID, err := resolveBefore(ctx, s.svc, col.ID, in.Top, in.Before, in.After)
	if err != nil {
		respondError(w, err)
		return
	}
	if err := s.svc.MoveTask(ctx, t.ID, col.ID, beforeID); err != nil {
		respondError(w, err)
		return
	}
	updated, err := s.svc.GetTask(ctx, t.ID)
	if err != nil {
		respondError(w, err)
		return
	}
	writeOK(w, http.StatusOK, updated)
}

func (s *Server) handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")
	if !parseBool(r.URL.Query().Get("force")) {
		writeErr(w, http.StatusConflict, "CONFLICT", fmt.Sprintf("refusing to delete task %q without ?force=true", id))
		return
	}
	t, err := s.svc.GetTaskByPrefix(ctx, id)
	if err != nil {
		respondError(w, err)
		return
	}
	if err := s.svc.DeleteTask(ctx, t.ID); err != nil {
		respondError(w, err)
		return
	}
	writeOK(w, http.StatusOK, map[string]any{"deleted": t.ID, "title": t.Title})
}

func findBoard(ctx context.Context, conn *sqlx.DB, projectName, boardName string) (*model.Board, []*model.Column, error) {
	p, err := db.GetProjectByName(ctx, conn, projectName)
	if err != nil {
		return nil, nil, err
	}
	boards, err := db.ListBoards(ctx, conn, p.ID)
	if err != nil {
		return nil, nil, err
	}
	var found *model.Board
	for _, b := range boards {
		if b.Name == boardName {
			found = b
			break
		}
	}
	if found == nil {
		return nil, nil, fmt.Errorf("%w: board %q in project %q", db.ErrNotFound, boardName, projectName)
	}
	cols, err := db.ListColumns(ctx, conn, found.ID)
	if err != nil {
		return nil, nil, err
	}
	return found, cols, nil
}

func findBoardByName(ctx context.Context, conn *sqlx.DB, boardName string) (*model.Board, []*model.Column, error) {
	projects, err := db.ListProjects(ctx, conn)
	if err != nil {
		return nil, nil, err
	}
	for _, p := range projects {
		boards, err := db.ListBoards(ctx, conn, p.ID)
		if err != nil {
			return nil, nil, err
		}
		for _, b := range boards {
			if b.Name == boardName {
				cols, err := db.ListColumns(ctx, conn, b.ID)
				if err != nil {
					return nil, nil, err
				}
				return b, cols, nil
			}
		}
	}
	return nil, nil, fmt.Errorf("%w: board %q", db.ErrNotFound, boardName)
}

func parseBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "t", "yes", "y":
		return true
	}
	return false
}

func buildTaskFilter(q url.Values) (db.TaskFilter, error) {
	filter := db.TaskFilter{
		ColumnName:      q.Get("column"),
		Search:          q.Get("search"),
		IncludeDone:     parseBool(q.Get("include_done")),
		IncludeArchived: parseBool(q.Get("include_archived")),
		OnlyArchived:    parseBool(q.Get("archived")),
	}
	if p := q.Get("priority"); p != "" {
		for _, piece := range strings.Split(p, ",") {
			piece = strings.TrimSpace(piece)
			if piece == "" {
				continue
			}
			pri, err := model.NewPriority(piece)
			if err != nil {
				return filter, err
			}
			filter.Priorities = append(filter.Priorities, pri)
		}
	}
	return filter, nil
}

func buildTaskPatch(ctx context.Context, in updateTaskReq, svc *service.Service) (db.TaskPatch, error) {
	patch := db.TaskPatch{}
	if in.Title != nil {
		v := *in.Title
		patch.Title = &v
	}
	if in.Desc != nil {
		v := *in.Desc
		patch.Description = &v
	}
	if in.Priority != nil {
		pri, err := model.NewPriority(*in.Priority)
		if err != nil {
			return patch, err
		}
		patch.Priority = &pri
	}
	if in.ClearDue {
		patch.ClearDueDate = true
	} else if in.Due != nil && *in.Due != "" {
		due, err := service.ParseDue(*in.Due)
		if err != nil {
			return patch, fmt.Errorf("due: %w", err)
		}
		patch.DueDate = due
	}
	if in.ClearParent {
		patch.ClearParentID = true
	} else if in.Parent != nil && *in.Parent != "" {
		t, err := svc.GetTaskByPrefix(ctx, *in.Parent)
		if err != nil {
			return patch, fmt.Errorf("parent: %w", err)
		}
		id := t.ID
		patch.ParentID = &id
	}
	return patch, nil
}

func resolveBefore(ctx context.Context, svc *service.Service, colID int64, top bool, before, after string) (*int64, error) {
	switch {
	case top:
		first, err := db.FirstTaskInColumn(ctx, svc.DB(), colID)
		if err != nil {
			return nil, err
		}
		if first == nil {
			return nil, nil
		}
		id := first.ID
		return &id, nil
	case before != "":
		t, err := svc.GetTaskByPrefix(ctx, before)
		if err != nil {
			return nil, err
		}
		id := t.ID
		return &id, nil
	case after != "":
		t, err := svc.GetTaskByPrefix(ctx, after)
		if err != nil {
			return nil, err
		}
		next, err := db.NextTaskInColumn(ctx, svc.DB(), colID, t.ID)
		if err != nil {
			return nil, err
		}
		if next == nil {
			return nil, nil
		}
		id := next.ID
		return &id, nil
	}
	return nil, nil
}
