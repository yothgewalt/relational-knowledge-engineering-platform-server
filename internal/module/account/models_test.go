package account

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestAccountJWTClaims_ToCustomClaims(t *testing.T) {
	tests := []struct {
		name     string
		claims   *AccountJWTClaims
		expected map[string]any
	}{
		{
			name: "valid claims",
			claims: &AccountJWTClaims{
				AccountID: "507f1f77bcf86cd799439011",
				Email:     "test@example.com",
				Username:  "testuser",
			},
			expected: map[string]any{
				"account_id": "507f1f77bcf86cd799439011",
				"email":      "test@example.com",
				"username":   "testuser",
			},
		},
		{
			name: "empty claims",
			claims: &AccountJWTClaims{
				AccountID: "",
				Email:     "",
				Username:  "",
			},
			expected: map[string]any{
				"account_id": "",
				"email":      "",
				"username":   "",
			},
		},
		{
			name: "partial claims",
			claims: &AccountJWTClaims{
				AccountID: "507f1f77bcf86cd799439011",
				Email:     "test@example.com",
				Username:  "",
			},
			expected: map[string]any{
				"account_id": "507f1f77bcf86cd799439011",
				"email":      "test@example.com",
				"username":   "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.claims.ToCustomClaims()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewAccountJWTClaimsFromCustom(t *testing.T) {
	tests := []struct {
		name         string
		customClaims map[string]any
		expected     *AccountJWTClaims
	}{
		{
			name: "valid custom claims",
			customClaims: map[string]any{
				"account_id": "507f1f77bcf86cd799439011",
				"email":      "test@example.com",
				"username":   "testuser",
			},
			expected: &AccountJWTClaims{
				AccountID: "507f1f77bcf86cd799439011",
				Email:     "test@example.com",
				Username:  "testuser",
			},
		},
		{
			name:         "empty custom claims",
			customClaims: map[string]any{},
			expected: &AccountJWTClaims{
				AccountID: "",
				Email:     "",
				Username:  "",
			},
		},
		{
			name: "partial custom claims",
			customClaims: map[string]any{
				"account_id": "507f1f77bcf86cd799439011",
				"email":      "test@example.com",
			},
			expected: &AccountJWTClaims{
				AccountID: "507f1f77bcf86cd799439011",
				Email:     "test@example.com",
				Username:  "",
			},
		},
		{
			name: "invalid types in custom claims",
			customClaims: map[string]any{
				"account_id": 12345, // not a string
				"email":      "test@example.com",
				"username":   []string{"test"}, // not a string
			},
			expected: &AccountJWTClaims{
				AccountID: "",
				Email:     "test@example.com",
				Username:  "",
			},
		},
		{
			name: "nil values in custom claims",
			customClaims: map[string]any{
				"account_id": nil,
				"email":      "test@example.com",
				"username":   nil,
			},
			expected: &AccountJWTClaims{
				AccountID: "",
				Email:     "test@example.com",
				Username:  "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewAccountJWTClaimsFromCustom(tt.customClaims)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAccount_ToResponse(t *testing.T) {
	objectID := primitive.NewObjectID()
	now := time.Now().Truncate(time.Millisecond)

	tests := []struct {
		name     string
		account  *Account
		expected *AccountResponse
	}{
		{
			name: "valid account conversion",
			account: &Account{
				ID:        objectID,
				Email:     "test@example.com",
				Username:  "testuser",
				FirstName: "Test",
				LastName:  "User",
				Avatar:    "https://example.com/avatar.jpg",
				IsActive:  true,
				CreatedAt: now,
				UpdatedAt: now,
			},
			expected: &AccountResponse{
				ID:        objectID.Hex(),
				Email:     "test@example.com",
				Username:  "testuser",
				FirstName: "Test",
				LastName:  "User",
				Avatar:    "https://example.com/avatar.jpg",
				IsActive:  true,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		{
			name: "account with empty fields",
			account: &Account{
				ID:        objectID,
				Email:     "",
				Username:  "",
				FirstName: "",
				LastName:  "",
				Avatar:    "",
				IsActive:  false,
				CreatedAt: now,
				UpdatedAt: now,
			},
			expected: &AccountResponse{
				ID:        objectID.Hex(),
				Email:     "",
				Username:  "",
				FirstName: "",
				LastName:  "",
				Avatar:    "",
				IsActive:  false,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		{
			name: "account with zero times",
			account: &Account{
				ID:        objectID,
				Email:     "test@example.com",
				Username:  "testuser",
				FirstName: "Test",
				LastName:  "User",
				Avatar:    "avatar.jpg",
				IsActive:  true,
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
			},
			expected: &AccountResponse{
				ID:        objectID.Hex(),
				Email:     "test@example.com",
				Username:  "testuser",
				FirstName: "Test",
				LastName:  "User",
				Avatar:    "avatar.jpg",
				IsActive:  true,
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.account.ToResponse()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAccountJSONMarshalUnmarshal(t *testing.T) {
	objectID := primitive.NewObjectID()
	now := time.Now()

	original := &Account{
		ID:        objectID,
		Email:     "test@example.com",
		Username:  "testuser",
		FirstName: "Test",
		LastName:  "User",
		Avatar:    "https://example.com/avatar.jpg",
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	jsonData, err := json.Marshal(original)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaled Account
	err = json.Unmarshal(jsonData, &unmarshaled)
	assert.NoError(t, err)

	assert.Equal(t, original.ID, unmarshaled.ID)
	assert.Equal(t, original.Email, unmarshaled.Email)
	assert.Equal(t, original.Username, unmarshaled.Username)
	assert.Equal(t, original.FirstName, unmarshaled.FirstName)
	assert.Equal(t, original.LastName, unmarshaled.LastName)
	assert.Equal(t, original.Avatar, unmarshaled.Avatar)
	assert.Equal(t, original.IsActive, unmarshaled.IsActive)
}

func TestCreateAccountRequestJSONMarshalUnmarshal(t *testing.T) {
	original := &CreateAccountRequest{
		Email:     "test@example.com",
		Username:  "testuser",
		FirstName: "Test",
		LastName:  "User",
		Avatar:    "https://example.com/avatar.jpg",
	}

	jsonData, err := json.Marshal(original)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaled CreateAccountRequest
	err = json.Unmarshal(jsonData, &unmarshaled)
	assert.NoError(t, err)

	assert.Equal(t, original, &unmarshaled)
}

func TestUpdateAccountRequestJSONMarshalUnmarshal(t *testing.T) {
	username := "updateduser"
	firstName := "Updated"
	isActive := false

	original := &UpdateAccountRequest{
		Username:  &username,
		FirstName: &firstName,
		LastName:  nil,
		Avatar:    StringPtr("https://example.com/new-avatar.jpg"),
		IsActive:  &isActive,
	}

	jsonData, err := json.Marshal(original)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaled UpdateAccountRequest
	err = json.Unmarshal(jsonData, &unmarshaled)
	assert.NoError(t, err)

	assert.Equal(t, original.Username, unmarshaled.Username)
	assert.Equal(t, original.FirstName, unmarshaled.FirstName)
	assert.Nil(t, unmarshaled.LastName)
	assert.Equal(t, original.Avatar, unmarshaled.Avatar)
	assert.Equal(t, original.IsActive, unmarshaled.IsActive)
}

func TestAccountResponseJSONMarshalUnmarshal(t *testing.T) {
	now := time.Now()

	original := &AccountResponse{
		ID:        primitive.NewObjectID().Hex(),
		Email:     "test@example.com",
		Username:  "testuser",
		FirstName: "Test",
		LastName:  "User",
		Avatar:    "https://example.com/avatar.jpg",
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	jsonData, err := json.Marshal(original)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaled AccountResponse
	err = json.Unmarshal(jsonData, &unmarshaled)
	assert.NoError(t, err)

	assert.Equal(t, original.ID, unmarshaled.ID)
	assert.Equal(t, original.Email, unmarshaled.Email)
	assert.Equal(t, original.Username, unmarshaled.Username)
	assert.Equal(t, original.FirstName, unmarshaled.FirstName)
	assert.Equal(t, original.LastName, unmarshaled.LastName)
	assert.Equal(t, original.Avatar, unmarshaled.Avatar)
	assert.Equal(t, original.IsActive, unmarshaled.IsActive)
}


func TestAccountJWTClaimsJSONMarshalUnmarshal(t *testing.T) {
	original := &AccountJWTClaims{
		AccountID: "507f1f77bcf86cd799439011",
		Email:     "test@example.com",
		Username:  "testuser",
	}

	jsonData, err := json.Marshal(original)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaled AccountJWTClaims
	err = json.Unmarshal(jsonData, &unmarshaled)
	assert.NoError(t, err)

	assert.Equal(t, original, &unmarshaled)
}

func TestAccountJWTClaimsRoundTrip(t *testing.T) {
	original := CreateTestAccountJWTClaims()

	customClaims := original.ToCustomClaims()

	result := NewAccountJWTClaimsFromCustom(customClaims)

	assert.Equal(t, original, result)
}

func TestCreateTestHelpers(t *testing.T) {
	t.Run("CreateTestAccount", func(t *testing.T) {
		account := CreateTestAccount()
		assert.NotEmpty(t, account.ID)
		assert.Equal(t, "test@example.com", account.Email)
		assert.Equal(t, "testuser", account.Username)
		assert.True(t, account.IsActive)

		customAccount := CreateTestAccount(func(a *Account) {
			a.Email = "custom@example.com"
			a.IsActive = false
		})
		assert.Equal(t, "custom@example.com", customAccount.Email)
		assert.False(t, customAccount.IsActive)
	})

	t.Run("CreateTestAccountResponse", func(t *testing.T) {
		response := CreateTestAccountResponse()
		assert.NotEmpty(t, response.ID)
		assert.Equal(t, "test@example.com", response.Email)
		assert.True(t, response.IsActive)
	})

	t.Run("CreateTestCreateAccountRequest", func(t *testing.T) {
		req := CreateTestCreateAccountRequest()
		assert.Equal(t, "test@example.com", req.Email)
		assert.Equal(t, "testuser", req.Username)
	})

	t.Run("CreateTestUpdateAccountRequest", func(t *testing.T) {
		req := CreateTestUpdateAccountRequest()
		assert.NotNil(t, req.Username)
		assert.Equal(t, "updateduser", *req.Username)
		assert.NotNil(t, req.IsActive)
		assert.True(t, *req.IsActive)
	})

	t.Run("Helper pointers", func(t *testing.T) {
		strPtr := StringPtr("test")
		assert.Equal(t, "test", *strPtr)

		boolPtr := BoolPtr(true)
		assert.True(t, *boolPtr)
	})
}
