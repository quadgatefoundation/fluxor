package auth

import (
	"testing"
	"time"
)

func TestGenerateToken(t *testing.T) {
	tests := []struct {
		name     string
		userID   int
		username string
		wantErr  bool
	}{
		{
			name:     "valid token generation",
			userID:   1,
			username: "testuser",
			wantErr:  false,
		},
		{
			name:     "token with different user",
			userID:   999,
			username: "anotheruser",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GenerateToken(tt.userID, tt.username)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if token == "" {
					t.Error("GenerateToken() returned empty token")
				}
			}
		})
	}
}

func TestValidateToken(t *testing.T) {
	// Generate a valid token
	userID := 1
	username := "testuser"
	token, err := GenerateToken(userID, username)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "valid token",
			token:   token,
			wantErr: false,
		},
		{
			name:    "invalid token",
			token:   "invalid.token.here",
			wantErr: true,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := ValidateToken(tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if claims == nil {
					t.Error("ValidateToken() returned nil claims")
					return
				}
				if claims.UserID != userID {
					t.Errorf("ValidateToken() UserID = %v, want %v", claims.UserID, userID)
				}
				if claims.Username != username {
					t.Errorf("ValidateToken() Username = %v, want %v", claims.Username, username)
				}
			}
		})
	}
}

func TestTokenExpiration(t *testing.T) {
	userID := 1
	username := "testuser"
	token, err := GenerateToken(userID, username)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Token should be valid immediately
	claims, err := ValidateToken(token)
	if err != nil {
		t.Fatalf("Token should be valid: %v", err)
	}

	// Check expiration is in the future
	if claims.ExpiresAt == nil {
		t.Fatal("Token claims should have expiration time")
	}
	if !claims.ExpiresAt.Time.After(time.Now()) {
		t.Error("Token expiration should be in the future")
	}
}

func TestTokenRoundTrip(t *testing.T) {
	userID := 123
	username := "roundtrip_user"

	// Generate token
	token, err := GenerateToken(userID, username)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Validate token
	claims, err := ValidateToken(token)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	// Verify claims match
	if claims.UserID != userID {
		t.Errorf("UserID mismatch: got %v, want %v", claims.UserID, userID)
	}
	if claims.Username != username {
		t.Errorf("Username mismatch: got %v, want %v", claims.Username, username)
	}
}
