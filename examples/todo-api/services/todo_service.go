package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/fluxorio/fluxor/examples/todo-api/models"
	"github.com/google/uuid"
)

// TodoServiceInterface defines the interface for todo service operations
type TodoServiceInterface interface {
	CreateTodo(ctx context.Context, userID uuid.UUID, req models.CreateTodoRequest) (*models.Todo, error)
	GetTodoByID(ctx context.Context, todoID, userID uuid.UUID) (*models.Todo, error)
	ListTodos(ctx context.Context, userID uuid.UUID, page, pageSize int, completed *bool) (*models.TodoListResponse, error)
	UpdateTodo(ctx context.Context, todoID, userID uuid.UUID, req models.UpdateTodoRequest) (*models.Todo, error)
	DeleteTodo(ctx context.Context, todoID, userID uuid.UUID) error
}

// TodoService handles todo-related operations
type TodoService struct {
	db *sql.DB
}

// NewTodoService creates a new todo service
func NewTodoService(db *sql.DB) *TodoService {
	return &TodoService{db: db}
}

// CreateTodo creates a new todo
func (s *TodoService) CreateTodo(ctx context.Context, userID uuid.UUID, req models.CreateTodoRequest) (*models.Todo, error) {
	todo := &models.Todo{
		ID:          uuid.New(),
		UserID:      userID,
		Title:       req.Title,
		Description: req.Description,
		Completed:   false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	query := `INSERT INTO todos (id, user_id, title, description, completed, created_at, updated_at) 
	          VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := s.db.ExecContext(ctx, query, todo.ID, todo.UserID, todo.Title, todo.Description, todo.Completed, todo.CreatedAt, todo.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create todo: %w", err)
	}

	return todo, nil
}

// GetTodoByID retrieves a todo by ID
func (s *TodoService) GetTodoByID(ctx context.Context, todoID, userID uuid.UUID) (*models.Todo, error) {
	todo := &models.Todo{}
	query := `SELECT id, user_id, title, description, completed, created_at, updated_at 
	          FROM todos WHERE id = $1 AND user_id = $2`
	err := s.db.QueryRowContext(ctx, query, todoID, userID).Scan(
		&todo.ID, &todo.UserID, &todo.Title, &todo.Description, &todo.Completed, &todo.CreatedAt, &todo.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("todo not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get todo: %w", err)
	}
	return todo, nil
}

// ListTodos retrieves a paginated list of todos for a user
func (s *TodoService) ListTodos(ctx context.Context, userID uuid.UUID, page, pageSize int, completed *bool) (*models.TodoListResponse, error) {
	offset := (page - 1) * pageSize

	// Build query
	baseQuery := `SELECT id, user_id, title, description, completed, created_at, updated_at FROM todos WHERE user_id = $1`
	countQuery := `SELECT COUNT(*) FROM todos WHERE user_id = $1`
	args := []interface{}{userID}
	argIndex := 2

	if completed != nil {
		baseQuery += fmt.Sprintf(" AND completed = $%d", argIndex)
		countQuery += fmt.Sprintf(" AND completed = $%d", argIndex)
		args = append(args, *completed)
		argIndex++
	}

	baseQuery += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, pageSize, offset)

	// Get total count
	var total int
	err := s.db.QueryRowContext(ctx, countQuery, args[:len(args)-2]...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count todos: %w", err)
	}

	// Get todos
	rows, err := s.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list todos: %w", err)
	}
	defer rows.Close()

	todos := []models.Todo{}
	for rows.Next() {
		todo := models.Todo{}
		err := rows.Scan(&todo.ID, &todo.UserID, &todo.Title, &todo.Description, &todo.Completed, &todo.CreatedAt, &todo.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan todo: %w", err)
		}
		todos = append(todos, todo)
	}

	totalPages := (total + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}

	return &models.TodoListResponse{
		Todos:      todos,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// UpdateTodo updates a todo
func (s *TodoService) UpdateTodo(ctx context.Context, todoID, userID uuid.UUID, req models.UpdateTodoRequest) (*models.Todo, error) {
	// Get existing todo first
	todo, err := s.GetTodoByID(ctx, todoID, userID)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.Title != nil {
		todo.Title = *req.Title
	}
	if req.Description != nil {
		todo.Description = *req.Description
	}
	if req.Completed != nil {
		todo.Completed = *req.Completed
	}
	todo.UpdatedAt = time.Now()

	query := `UPDATE todos SET title = $1, description = $2, completed = $3, updated_at = $4 
	          WHERE id = $5 AND user_id = $6`
	_, err = s.db.ExecContext(ctx, query, todo.Title, todo.Description, todo.Completed, todo.UpdatedAt, todoID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to update todo: %w", err)
	}

	return todo, nil
}

// DeleteTodo deletes a todo
func (s *TodoService) DeleteTodo(ctx context.Context, todoID, userID uuid.UUID) error {
	query := `DELETE FROM todos WHERE id = $1 AND user_id = $2`
	result, err := s.db.ExecContext(ctx, query, todoID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete todo: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("todo not found")
	}

	return nil
}
