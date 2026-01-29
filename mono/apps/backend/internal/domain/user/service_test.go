package user

import (
	"context"
	"testing"
	"time"
)

// MockUserRepository is a mock implementation of Repository for testing
type MockUserRepository struct {
	users       map[string]*User
	apiKeys     map[string]*APIKey
	keysByHash  map[string]*APIKey
	createErr   error
	findErr     error
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users:      make(map[string]*User),
		apiKeys:    make(map[string]*APIKey),
		keysByHash: make(map[string]*APIKey),
	}
}

func (m *MockUserRepository) Create(ctx context.Context, user *User) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.users[user.ID] = user
	return nil
}

func (m *MockUserRepository) FindByID(ctx context.Context, id string) (*User, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	user, ok := m.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, ErrUserNotFound
}

func (m *MockUserRepository) FindByUsername(ctx context.Context, username string) (*User, error) {
	for _, user := range m.users {
		if user.Username == username {
			return user, nil
		}
	}
	return nil, ErrUserNotFound
}

func (m *MockUserRepository) Update(ctx context.Context, user *User) error {
	if _, ok := m.users[user.ID]; !ok {
		return ErrUserNotFound
	}
	m.users[user.ID] = user
	return nil
}

func (m *MockUserRepository) Delete(ctx context.Context, id string) error {
	delete(m.users, id)
	return nil
}

func (m *MockUserRepository) CreateAPIKey(ctx context.Context, key *APIKey) error {
	m.apiKeys[key.ID] = key
	m.keysByHash[key.KeyHash] = key
	return nil
}

func (m *MockUserRepository) FindAPIKeyByHash(ctx context.Context, hash string) (*APIKey, error) {
	key, ok := m.keysByHash[hash]
	if !ok {
		return nil, ErrAPIKeyNotFound
	}
	return key, nil
}

func (m *MockUserRepository) ListAPIKeys(ctx context.Context, userID string) ([]*APIKey, error) {
	var keys []*APIKey
	for _, key := range m.apiKeys {
		if key.UserID == userID {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

func (m *MockUserRepository) DeleteAPIKey(ctx context.Context, userID, keyID string) error {
	if key, ok := m.apiKeys[keyID]; ok && key.UserID == userID {
		delete(m.keysByHash, key.KeyHash)
		delete(m.apiKeys, keyID)
		return nil
	}
	return ErrAPIKeyNotFound
}

func (m *MockUserRepository) SaveRefreshToken(ctx context.Context, token *RefreshToken) error {
	return nil
}

func (m *MockUserRepository) FindRefreshToken(ctx context.Context, token string) (*RefreshToken, error) {
	return nil, nil
}

func (m *MockUserRepository) DeleteRefreshToken(ctx context.Context, token string) error {
	return nil
}

func (m *MockUserRepository) DeleteUserRefreshTokens(ctx context.Context, userID string) error {
	return nil
}

func TestService_Register(t *testing.T) {
	repo := NewMockUserRepository()
	svc := NewService(repo, ServiceConfig{
		JWTSecret:          "test-secret-key-for-jwt-signing",
		JWTExpiration:      time.Hour,
		RefreshExpiration:  24 * time.Hour,
		AllowRegistration:  true,
		RequireEmailVerify: false,
	})

	tests := []struct {
		name    string
		req     RegisterRequest
		wantErr bool
	}{
		{
			name: "successful registration",
			req: RegisterRequest{
				Email:    "test@example.com",
				Password: "password123",
				Username: "testuser",
			},
			wantErr: false,
		},
		{
			name: "missing email",
			req: RegisterRequest{
				Password: "password123",
				Username: "testuser2",
			},
			wantErr: true,
		},
		{
			name: "missing password",
			req: RegisterRequest{
				Email:    "test2@example.com",
				Username: "testuser3",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Register(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestService_Login(t *testing.T) {
	repo := NewMockUserRepository()
	svc := NewService(repo, ServiceConfig{
		JWTSecret:         "test-secret-key-for-jwt-signing",
		JWTExpiration:     time.Hour,
		RefreshExpiration: 24 * time.Hour,
		AllowRegistration: true,
	})

	// Register a user first
	_, err := svc.Register(context.Background(), RegisterRequest{
		Email:    "login@example.com",
		Password: "password123",
		Username: "loginuser",
	})
	if err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}

	tests := []struct {
		name    string
		req     LoginRequest
		wantErr bool
	}{
		{
			name: "successful login with email",
			req: LoginRequest{
				Email:    "login@example.com",
				Password: "password123",
			},
			wantErr: false,
		},
		{
			name: "successful login with username",
			req: LoginRequest{
				Username: "loginuser",
				Password: "password123",
			},
			wantErr: false,
		},
		{
			name: "wrong password",
			req: LoginRequest{
				Email:    "login@example.com",
				Password: "wrongpassword",
			},
			wantErr: true,
		},
		{
			name: "non-existent user",
			req: LoginRequest{
				Email:    "nonexistent@example.com",
				Password: "password123",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Login(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.Login() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestService_CreateAPIKey(t *testing.T) {
	repo := NewMockUserRepository()
	svc := NewService(repo, ServiceConfig{
		JWTSecret:         "test-secret-key-for-jwt-signing",
		JWTExpiration:     time.Hour,
		RefreshExpiration: 24 * time.Hour,
		AllowRegistration: true,
	})

	// Register a user
	authResp, err := svc.Register(context.Background(), RegisterRequest{
		Email:    "apikey@example.com",
		Password: "password123",
		Username: "apikeyuser",
	})
	if err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}

	// Create API key
	key, rawKey, err := svc.CreateAPIKey(context.Background(), authResp.User.ID, "Test Key", nil)
	if err != nil {
		t.Fatalf("Failed to create API key: %v", err)
	}

	if key.Name != "Test Key" {
		t.Errorf("Expected key name 'Test Key', got '%s'", key.Name)
	}

	if rawKey == "" {
		t.Error("Expected raw key to be returned")
	}

	// Validate API key
	validatedKey, err := svc.ValidateAPIKey(context.Background(), rawKey)
	if err != nil {
		t.Errorf("Failed to validate API key: %v", err)
	}

	if validatedKey.ID != key.ID {
		t.Errorf("Expected key ID %s, got %s", key.ID, validatedKey.ID)
	}
}

func TestService_ValidateToken(t *testing.T) {
	repo := NewMockUserRepository()
	svc := NewService(repo, ServiceConfig{
		JWTSecret:         "test-secret-key-for-jwt-signing",
		JWTExpiration:     time.Hour,
		RefreshExpiration: 24 * time.Hour,
		AllowRegistration: true,
	})

	// Register and login to get a token
	authResp, err := svc.Register(context.Background(), RegisterRequest{
		Email:    "validate@example.com",
		Password: "password123",
		Username: "validateuser",
	})
	if err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}

	// Validate the token
	claims, err := svc.ValidateToken(context.Background(), authResp.AccessToken)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if claims.UserID != authResp.User.ID {
		t.Errorf("Expected user ID %s, got %s", authResp.User.ID, claims.UserID)
	}

	// Test invalid token
	_, err = svc.ValidateToken(context.Background(), "invalid-token")
	if err == nil {
		t.Error("Expected error for invalid token")
	}
}
