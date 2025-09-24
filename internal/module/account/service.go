package account

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/mongo"
)

type AccountService interface {
	CreateAccount(ctx context.Context, req *CreateAccountRequest) (*AccountResponse, error)
	GetAccountByID(ctx context.Context, id string) (*AccountResponse, error)
	GetAccountByEmail(ctx context.Context, email string) (*AccountResponse, error)
	GetAccountByUsername(ctx context.Context, username string) (*AccountResponse, error)
	UpdateAccount(ctx context.Context, id string, req *UpdateAccountRequest) (*AccountResponse, error)
	DeleteAccount(ctx context.Context, id string) error
	ListAccounts(ctx context.Context, req *ListAccountsRequest) (*mongo.PaginatedResult[AccountResponse], error)
}

type accountService struct {
	repository AccountRepository
}

func NewAccountService(mongoService *mongo.MongoService) AccountService {
	repository := NewAccountRepository(mongoService)
	
	return &accountService{
		repository: repository,
	}
}

func (s *accountService) CreateAccount(ctx context.Context, req *CreateAccountRequest) (*AccountResponse, error) {
	exists, err := s.repository.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check email existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("account with email %s already exists", req.Email)
	}

	exists, err = s.repository.ExistsByUsername(ctx, req.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to check username existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("account with username %s already exists", req.Username)
	}

	account := &Account{
		Email:     req.Email,
		Username:  req.Username,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Avatar:    req.Avatar,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	createdAccount, err := s.repository.Create(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	return createdAccount.ToResponse(), nil
}

func (s *accountService) GetAccountByID(ctx context.Context, id string) (*AccountResponse, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid account ID format: %w", err)
	}

	account, err := s.repository.GetByID(ctx, objectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	if account == nil {
		return nil, fmt.Errorf("account not found")
	}

	return account.ToResponse(), nil
}

func (s *accountService) GetAccountByEmail(ctx context.Context, email string) (*AccountResponse, error) {
	account, err := s.repository.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get account by email: %w", err)
	}

	if account == nil {
		return nil, fmt.Errorf("account not found")
	}

	return account.ToResponse(), nil
}

func (s *accountService) GetAccountByUsername(ctx context.Context, username string) (*AccountResponse, error) {
	account, err := s.repository.GetByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get account by username: %w", err)
	}

	if account == nil {
		return nil, fmt.Errorf("account not found")
	}

	return account.ToResponse(), nil
}

func (s *accountService) UpdateAccount(ctx context.Context, id string, req *UpdateAccountRequest) (*AccountResponse, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid account ID format: %w", err)
	}

	updateData := bson.M{
		"updated_at": time.Now(),
	}

	if req.Username != nil && *req.Username != "" {
		exists, err := s.repository.ExistsByUsername(ctx, *req.Username)
		if err != nil {
			return nil, fmt.Errorf("failed to check username existence: %w", err)
		}
		
		if exists {
			existing, err := s.repository.GetByUsername(ctx, *req.Username)
			if err != nil {
				return nil, fmt.Errorf("failed to get existing account: %w", err)
			}
			if existing != nil && existing.ID != objectID {
				return nil, fmt.Errorf("username %s is already taken", *req.Username)
			}
		}
		updateData["username"] = *req.Username
	}

	if req.FirstName != nil {
		updateData["first_name"] = *req.FirstName
	}

	if req.LastName != nil {
		updateData["last_name"] = *req.LastName
	}

	if req.Avatar != nil {
		updateData["avatar"] = *req.Avatar
	}

	if req.IsActive != nil {
		updateData["is_active"] = *req.IsActive
	}

	updatedAccount, err := s.repository.Update(ctx, objectID, updateData)
	if err != nil {
		return nil, fmt.Errorf("failed to update account: %w", err)
	}

	if updatedAccount == nil {
		return nil, fmt.Errorf("account not found")
	}

	return updatedAccount.ToResponse(), nil
}

func (s *accountService) DeleteAccount(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid account ID format: %w", err)
	}

	account, err := s.repository.GetByID(ctx, objectID)
	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}

	if account == nil {
		return fmt.Errorf("account not found")
	}

	err = s.repository.Delete(ctx, objectID)
	if err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}

	return nil
}

func (s *accountService) ListAccounts(ctx context.Context, req *ListAccountsRequest) (*mongo.PaginatedResult[AccountResponse], error) {
	filter := bson.M{}

	if req.Email != "" {
		filter["email"] = bson.M{"$regex": req.Email, "$options": "i"}
	}

	if req.Username != "" {
		filter["username"] = bson.M{"$regex": req.Username, "$options": "i"}
	}

	if req.IsActive != nil {
		filter["is_active"] = *req.IsActive
	}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}

	pagination := mongo.PaginationOptions{
		Page:  req.Page,
		Limit: req.Limit,
	}

	result, err := s.repository.List(ctx, filter, pagination)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}

	responses := make([]AccountResponse, len(result.Data))
	for i, account := range result.Data {
		responses[i] = *account.ToResponse()
	}

	return &mongo.PaginatedResult[AccountResponse]{
		Data:       responses,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
		HasNext:    result.HasNext,
		HasPrev:    result.HasPrev,
	}, nil
}

