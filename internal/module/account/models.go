package account

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Account struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Email        string             `json:"email" bson:"email"`
	Username     string             `json:"username" bson:"username"`
	FirstName    string             `json:"first_name" bson:"first_name"`
	LastName     string             `json:"last_name" bson:"last_name"`
	Avatar       string             `json:"avatar" bson:"avatar"`
	PasswordHash string             `json:"-" bson:"password_hash"`
	IsActive     bool               `json:"is_active" bson:"is_active"`
	CreatedAt    time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at" bson:"updated_at"`
}

type CreateAccountRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Username  string `json:"username" validate:"required,min=3,max=50"`
	FirstName string `json:"first_name" validate:"required,min=1,max=100"`
	LastName  string `json:"last_name" validate:"required,min=1,max=100"`
	Avatar    string `json:"avatar"`
}

type UpdateAccountRequest struct {
	Username  *string `json:"username,omitempty" validate:"omitempty,min=3,max=50"`
	FirstName *string `json:"first_name,omitempty" validate:"omitempty,min=1,max=100"`
	LastName  *string `json:"last_name,omitempty" validate:"omitempty,min=1,max=100"`
	Avatar    *string `json:"avatar,omitempty"`
	IsActive  *bool   `json:"is_active,omitempty"`
}

type AccountResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Avatar    string    `json:"avatar"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ListAccountsRequest struct {
	Page     int64  `query:"page" validate:"min=1"`
	Limit    int64  `query:"limit" validate:"min=1,max=100"`
	Email    string `query:"email"`
	Username string `query:"username"`
	IsActive *bool  `query:"is_active"`
}

type AccountJWTClaims struct {
	AccountID string `json:"account_id"`
	Email     string `json:"email"`
	Username  string `json:"username"`
}

func (c *AccountJWTClaims) ToCustomClaims() map[string]any {
	return map[string]any{
		"account_id": c.AccountID,
		"email":      c.Email,
		"username":   c.Username,
	}
}

func NewAccountJWTClaimsFromCustom(customClaims map[string]any) *AccountJWTClaims {
	claims := &AccountJWTClaims{}

	if accountID, ok := customClaims["account_id"].(string); ok {
		claims.AccountID = accountID
	}

	if email, ok := customClaims["email"].(string); ok {
		claims.Email = email
	}

	if username, ok := customClaims["username"].(string); ok {
		claims.Username = username
	}

	return claims
}

func (a *Account) ToResponse() *AccountResponse {
	return &AccountResponse{
		ID:        a.ID.Hex(),
		Email:     a.Email,
		Username:  a.Username,
		FirstName: a.FirstName,
		LastName:  a.LastName,
		Avatar:    a.Avatar,
		IsActive:  a.IsActive,
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
	}
}
