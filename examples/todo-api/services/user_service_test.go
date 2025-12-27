package services

import (
	"context"
	"database/sql"
	"testing"

	"github.com/fluxorio/fluxor/examples/todo-api/models"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

func setupUserTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create users table
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT NOT NULL UNIQUE,
		email TEXT NOT NULL,
		password_hash TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	return db
}

func TestNewUserService(t *testing.T) {
	db := setupUserTestDB(t)
	defer db.Close()

	service := NewUserService(db)
	if service == nil {
		t.Fatal("NewUserService() returned nil")
	}
	if service.db != db {
		t.Error("NewUserService() did not set db correctly")
	}
}

func TestUserService_CreateUser(t *testing.T) {
	db := setupUserTestDB(t)
	defer db.Close()

	service := NewUserService(db)
	ctx := context.Background()

	tests := []struct {
		name    string
		req     models.CreateUserRequest
		wantErr bool
	}{
		{
			name: "valid user",
			req: models.CreateUserRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
			},
			wantErr: false,
		},
		{
			name: "user with special characters in username",
			req: models.CreateUserRequest{
				Username: "test_user_123",
				Email:    "test2@example.com",
				Password: "password123",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := service.CreateUser(ctx, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if user == nil {
					t.Fatal("CreateUser() returned nil user")
				}
				if user.Username != tt.req.Username {
					t.Errorf("CreateUser() Username = %v, want %v", user.Username, tt.req.Username)
				}
				if user.Email != tt.req.Email {
					t.Errorf("CreateUser() Email = %v, want %v", user.Email, tt.req.Email)
				}
				if user.PasswordHash == "" {
					t.Error("CreateUser() PasswordHash should not be empty")
				}
				// Verify password is hashed
				err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(tt.req.Password))
				if err != nil {
					t.Error("CreateUser() password should be properly hashed")
				}
			}
		})
	}
}

func TestUserService_GetUserByUsername(t *testing.T) {
	db := setupUserTestDB(t)
	defer db.Close()

	service := NewUserService(db)
	ctx := context.Background()

	// Create a user first
	req := models.CreateUserRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}
	createdUser, err := service.CreateUser(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	tests := []struct {
		name     string
		username string
		wantErr  bool
	}{
		{
			name:     "existing user",
			username: "testuser",
			wantErr:  false,
		},
		{
			name:     "non-existent user",
			username: "nonexistent",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := service.GetUserByUsername(ctx, tt.username)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserByUsername() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if user == nil {
					t.Fatal("GetUserByUsername() returned nil")
				}
				if user.ID != createdUser.ID {
					t.Errorf("GetUserByUsername() ID = %v, want %v", user.ID, createdUser.ID)
				}
				if user.Username != tt.username {
					t.Errorf("GetUserByUsername() Username = %v, want %v", user.Username, tt.username)
				}
			}
		})
	}
}

func TestUserService_GetUserByID(t *testing.T) {
	db := setupUserTestDB(t)
	defer db.Close()

	service := NewUserService(db)
	ctx := context.Background()

	// Create a user first
	req := models.CreateUserRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}
	createdUser, err := service.CreateUser(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	tests := []struct {
		name    string
		userID  uuid.UUID
		wantErr bool
	}{
		{
			name:    "existing user",
			userID:  createdUser.ID,
			wantErr: false,
		},
		{
			name:    "non-existent user",
			userID:  uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := service.GetUserByID(ctx, tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if user == nil {
					t.Fatal("GetUserByID() returned nil")
				}
				if user.ID != tt.userID {
					t.Errorf("GetUserByID() ID = %v, want %v", user.ID, tt.userID)
				}
			}
		})
	}
}

func TestUserService_VerifyPassword(t *testing.T) {
	service := NewUserService(nil) // DB not needed for this method

	tests := []struct {
		name          string
		password      string
		wrongPassword string
		wantMatch     bool
	}{
		{
			name:          "correct password",
			password:      "password123",
			wrongPassword: "password123",
			wantMatch:     true,
		},
		{
			name:          "wrong password",
			password:      "password123",
			wrongPassword: "wrongpassword",
			wantMatch:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Hash the password
			hash, err := bcrypt.GenerateFromPassword([]byte(tt.password), bcrypt.DefaultCost)
			if err != nil {
				t.Fatalf("Failed to hash password: %v", err)
			}

			result := service.VerifyPassword(string(hash), tt.wrongPassword)
			if result != tt.wantMatch {
				t.Errorf("VerifyPassword() = %v, want %v", result, tt.wantMatch)
			}
		})
	}
}

func TestGenerateSecretKey(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "generate secret key",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := GenerateSecretKey()
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateSecretKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if key == "" {
					t.Error("GenerateSecretKey() returned empty key")
				}
				if len(key) < 32 {
					t.Errorf("GenerateSecretKey() key length = %v, want >= 32", len(key))
				}
			}
		})
	}

	// Test that each call generates a different key
	key1, _ := GenerateSecretKey()
	key2, _ := GenerateSecretKey()
	if key1 == key2 {
		t.Error("GenerateSecretKey() should generate different keys each time")
	}
}
