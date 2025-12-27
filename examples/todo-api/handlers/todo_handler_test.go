package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/fluxorio/fluxor/examples/todo-api/models"
	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
)

// mockTodoService is a mock implementation of TodoService for testing
type mockTodoService struct {
	todos       map[uuid.UUID]*models.Todo
	createError error
	getError    error
	listError   error
	updateError error
	deleteError error
}

func (m *mockTodoService) CreateTodo(ctx context.Context, userID uuid.UUID, req models.CreateTodoRequest) (*models.Todo, error) {
	if m.createError != nil {
		return nil, m.createError
	}
	todo := &models.Todo{
		ID:          uuid.New(),
		UserID:      userID,
		Title:       req.Title,
		Description: req.Description,
		Completed:   false,
	}
	m.todos[todo.ID] = todo
	return todo, nil
}

func (m *mockTodoService) GetTodoByID(ctx context.Context, todoID, userID uuid.UUID) (*models.Todo, error) {
	if m.getError != nil {
		return nil, m.getError
	}
	todo, ok := m.todos[todoID]
	if !ok || todo.UserID != userID {
		return nil, errors.New("todo not found")
	}
	return todo, nil
}

func (m *mockTodoService) ListTodos(ctx context.Context, userID uuid.UUID, page, pageSize int, completed *bool) (*models.TodoListResponse, error) {
	if m.listError != nil {
		return nil, m.listError
	}
	todos := []models.Todo{}
	for _, todo := range m.todos {
		if todo.UserID == userID {
			if completed == nil || todo.Completed == *completed {
				todos = append(todos, *todo)
			}
		}
	}
	return &models.TodoListResponse{
		Todos:      todos,
		Total:      len(todos),
		Page:       page,
		PageSize:   pageSize,
		TotalPages: (len(todos) + pageSize - 1) / pageSize,
	}, nil
}

func (m *mockTodoService) UpdateTodo(ctx context.Context, todoID, userID uuid.UUID, req models.UpdateTodoRequest) (*models.Todo, error) {
	if m.updateError != nil {
		return nil, m.updateError
	}
	todo, ok := m.todos[todoID]
	if !ok || todo.UserID != userID {
		return nil, errors.New("todo not found")
	}
	if req.Title != nil {
		todo.Title = *req.Title
	}
	if req.Description != nil {
		todo.Description = *req.Description
	}
	if req.Completed != nil {
		todo.Completed = *req.Completed
	}
	return todo, nil
}

func (m *mockTodoService) DeleteTodo(ctx context.Context, todoID, userID uuid.UUID) error {
	if m.deleteError != nil {
		return m.deleteError
	}
	todo, ok := m.todos[todoID]
	if !ok || todo.UserID != userID {
		return errors.New("todo not found")
	}
	delete(m.todos, todoID)
	return nil
}

func createTestContext(t *testing.T, userID string) (*web.FastRequestContext, func()) {
	t.Helper()
	gocmd := core.NewGoCMD(context.Background())
	fasthttpCtx := &fasthttp.RequestCtx{}

	reqCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         fasthttpCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	if userID != "" {
		claims := jwt.MapClaims{
			"user_id": userID,
			"sub":     userID,
		}
		reqCtx.Set("user", claims)
	}

	cleanup := func() {
		gocmd.Close()
	}

	return reqCtx, cleanup
}

func TestTodoHandler_CreateTodo(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		body       map[string]interface{}
		wantStatus int
	}{
		{
			name:   "valid todo",
			userID: uuid.New().String(),
			body: map[string]interface{}{
				"title":       "Test Todo",
				"description": "Test Description",
			},
			wantStatus: 201,
		},
		{
			name:   "missing title",
			userID: uuid.New().String(),
			body: map[string]interface{}{
				"description": "Test Description",
			},
			wantStatus: 400,
		},
		{
			name:       "no user",
			userID:     "",
			body:       map[string]interface{}{"title": "Test"},
			wantStatus: 401,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockTodoService{todos: make(map[uuid.UUID]*models.Todo)}
			handler := NewTodoHandler(service)

			reqCtx, cleanup := createTestContext(t, tt.userID)
			defer cleanup()

			// Set request body
			bodyBytes, _ := json.Marshal(tt.body)
			reqCtx.RequestCtx.Request.SetBody(bodyBytes)
			reqCtx.RequestCtx.Request.Header.SetMethod("POST")
			reqCtx.RequestCtx.Request.Header.SetContentType("application/json")

			err := handler.CreateTodo(reqCtx)
			if err != nil {
				t.Errorf("CreateTodo() error = %v", err)
			}

			if reqCtx.RequestCtx.Response.StatusCode() != tt.wantStatus {
				t.Errorf("CreateTodo() status = %v, want %v", reqCtx.RequestCtx.Response.StatusCode(), tt.wantStatus)
			}
		})
	}
}

func TestTodoHandler_GetTodo(t *testing.T) {
	service := &mockTodoService{todos: make(map[uuid.UUID]*models.Todo)}
	handler := NewTodoHandler(service)
	userID := uuid.New()

	// Create a todo
	todo, _ := service.CreateTodo(context.Background(), userID, models.CreateTodoRequest{
		Title: "Test Todo",
	})

	reqCtx, cleanup := createTestContext(t, userID.String())
	defer cleanup()

	reqCtx.Params["id"] = todo.ID.String()
	reqCtx.RequestCtx.Request.Header.SetMethod("GET")

	err := handler.GetTodo(reqCtx)
	if err != nil {
		t.Errorf("GetTodo() error = %v", err)
	}

	if reqCtx.RequestCtx.Response.StatusCode() != 200 {
		t.Errorf("GetTodo() status = %v, want 200", reqCtx.RequestCtx.Response.StatusCode())
	}
}

func TestTodoHandler_ListTodos(t *testing.T) {
	service := &mockTodoService{todos: make(map[uuid.UUID]*models.Todo)}
	handler := NewTodoHandler(service)
	userID := uuid.New()

	// Create some todos
	for i := 0; i < 3; i++ {
		_, _ = service.CreateTodo(context.Background(), userID, models.CreateTodoRequest{
			Title: "Todo " + string(rune('0'+i)),
		})
	}

	reqCtx, cleanup := createTestContext(t, userID.String())
	defer cleanup()

	reqCtx.RequestCtx.Request.Header.SetMethod("GET")

	err := handler.ListTodos(reqCtx)
	if err != nil {
		t.Errorf("ListTodos() error = %v", err)
	}

	if reqCtx.RequestCtx.Response.StatusCode() != 200 {
		t.Errorf("ListTodos() status = %v, want 200", reqCtx.RequestCtx.Response.StatusCode())
	}
}
