package services

import (
	"context"
	"database/sql"
	"testing"

	"github.com/fluxorio/fluxor/examples/todo-api/models"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create tables
	schema := `
	CREATE TABLE IF NOT EXISTS todos (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		title TEXT NOT NULL,
		description TEXT,
		completed BOOLEAN NOT NULL DEFAULT 0,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	return db
}

func TestNewTodoService(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewTodoService(db)
	if service == nil {
		t.Fatal("NewTodoService() returned nil")
	}
	if service.db != db {
		t.Error("NewTodoService() did not set db correctly")
	}
}

func TestTodoService_CreateTodo(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewTodoService(db)
	ctx := context.Background()
	userID := uuid.New()

	tests := []struct {
		name    string
		userID  uuid.UUID
		req     models.CreateTodoRequest
		wantErr bool
	}{
		{
			name:   "valid todo",
			userID: userID,
			req: models.CreateTodoRequest{
				Title:       "Test Todo",
				Description: "Test Description",
			},
			wantErr: false,
		},
		{
			name:   "todo without description",
			userID: userID,
			req: models.CreateTodoRequest{
				Title: "Test Todo",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			todo, err := service.CreateTodo(ctx, tt.userID, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateTodo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if todo == nil {
					t.Fatal("CreateTodo() returned nil todo")
				}
				if todo.Title != tt.req.Title {
					t.Errorf("CreateTodo() Title = %v, want %v", todo.Title, tt.req.Title)
				}
				if todo.UserID != tt.userID {
					t.Errorf("CreateTodo() UserID = %v, want %v", todo.UserID, tt.userID)
				}
				if todo.Completed != false {
					t.Error("CreateTodo() Completed should be false")
				}
			}
		})
	}
}

func TestTodoService_GetTodoByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewTodoService(db)
	ctx := context.Background()
	userID := uuid.New()

	// Create a todo first
	todo, err := service.CreateTodo(ctx, userID, models.CreateTodoRequest{
		Title: "Test Todo",
	})
	if err != nil {
		t.Fatalf("Failed to create todo: %v", err)
	}

	tests := []struct {
		name    string
		todoID  uuid.UUID
		userID  uuid.UUID
		wantErr bool
	}{
		{
			name:    "existing todo",
			todoID:  todo.ID,
			userID:  userID,
			wantErr: false,
		},
		{
			name:    "non-existent todo",
			todoID:  uuid.New(),
			userID:  userID,
			wantErr: true,
		},
		{
			name:    "wrong user",
			todoID:  todo.ID,
			userID:  uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.GetTodoByID(ctx, tt.todoID, tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTodoByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if result == nil {
					t.Fatal("GetTodoByID() returned nil")
				}
				if result.ID != tt.todoID {
					t.Errorf("GetTodoByID() ID = %v, want %v", result.ID, tt.todoID)
				}
			}
		})
	}
}

func TestTodoService_ListTodos(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewTodoService(db)
	ctx := context.Background()
	userID := uuid.New()

	// Create some todos
	for i := 0; i < 5; i++ {
		_, err := service.CreateTodo(ctx, userID, models.CreateTodoRequest{
			Title:       "Todo " + string(rune('0'+i)),
			Description: "Description",
		})
		if err != nil {
			t.Fatalf("Failed to create todo: %v", err)
		}
	}

	tests := []struct {
		name      string
		userID    uuid.UUID
		page      int
		pageSize  int
		completed *bool
		wantErr   bool
	}{
		{
			name:     "list first page",
			userID:   userID,
			page:     1,
			pageSize: 3,
			wantErr:  false,
		},
		{
			name:     "list second page",
			userID:   userID,
			page:     2,
			pageSize: 3,
			wantErr:  false,
		},
		{
			name:     "empty result for different user",
			userID:   uuid.New(),
			page:     1,
			pageSize: 10,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.ListTodos(ctx, tt.userID, tt.page, tt.pageSize, tt.completed)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListTodos() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if result == nil {
					t.Fatal("ListTodos() returned nil")
				}
				if result.Page != tt.page {
					t.Errorf("ListTodos() Page = %v, want %v", result.Page, tt.page)
				}
				if result.PageSize != tt.pageSize {
					t.Errorf("ListTodos() PageSize = %v, want %v", result.PageSize, tt.pageSize)
				}
			}
		})
	}
}

func TestTodoService_UpdateTodo(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewTodoService(db)
	ctx := context.Background()
	userID := uuid.New()

	// Create a todo first
	todo, err := service.CreateTodo(ctx, userID, models.CreateTodoRequest{
		Title: "Original Title",
	})
	if err != nil {
		t.Fatalf("Failed to create todo: %v", err)
	}

	title := "Updated Title"
	description := "Updated Description"
	completed := true

	tests := []struct {
		name    string
		todoID  uuid.UUID
		userID  uuid.UUID
		req     models.UpdateTodoRequest
		wantErr bool
	}{
		{
			name:   "update title",
			todoID: todo.ID,
			userID: userID,
			req: models.UpdateTodoRequest{
				Title: &title,
			},
			wantErr: false,
		},
		{
			name:   "update completed",
			todoID: todo.ID,
			userID: userID,
			req: models.UpdateTodoRequest{
				Completed: &completed,
			},
			wantErr: false,
		},
		{
			name:   "update all fields",
			todoID: todo.ID,
			userID: userID,
			req: models.UpdateTodoRequest{
				Title:       &title,
				Description: &description,
				Completed:   &completed,
			},
			wantErr: false,
		},
		{
			name:   "non-existent todo",
			todoID: uuid.New(),
			userID: userID,
			req: models.UpdateTodoRequest{
				Title: &title,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.UpdateTodo(ctx, tt.todoID, tt.userID, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateTodo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if result == nil {
					t.Fatal("UpdateTodo() returned nil")
				}
				if tt.req.Title != nil && result.Title != *tt.req.Title {
					t.Errorf("UpdateTodo() Title = %v, want %v", result.Title, *tt.req.Title)
				}
				if tt.req.Completed != nil && result.Completed != *tt.req.Completed {
					t.Errorf("UpdateTodo() Completed = %v, want %v", result.Completed, *tt.req.Completed)
				}
			}
		})
	}
}

func TestTodoService_DeleteTodo(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewTodoService(db)
	ctx := context.Background()
	userID := uuid.New()

	// Create a todo first
	todo, err := service.CreateTodo(ctx, userID, models.CreateTodoRequest{
		Title: "To Delete",
	})
	if err != nil {
		t.Fatalf("Failed to create todo: %v", err)
	}

	tests := []struct {
		name    string
		todoID  uuid.UUID
		userID  uuid.UUID
		wantErr bool
	}{
		{
			name:    "delete existing todo",
			todoID:  todo.ID,
			userID:  userID,
			wantErr: false,
		},
		{
			name:    "delete non-existent todo",
			todoID:  uuid.New(),
			userID:  userID,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.DeleteTodo(ctx, tt.todoID, tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteTodo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
