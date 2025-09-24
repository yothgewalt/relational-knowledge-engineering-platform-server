package account

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/mongo"
)

const CollectionName = "accounts"

type AccountRepository interface {
	Create(ctx context.Context, account *Account) (*Account, error)
	GetByID(ctx context.Context, id primitive.ObjectID) (*Account, error)
	GetByEmail(ctx context.Context, email string) (*Account, error)
	GetByUsername(ctx context.Context, username string) (*Account, error)
	Update(ctx context.Context, id primitive.ObjectID, updateData bson.M) (*Account, error)
	Delete(ctx context.Context, id primitive.ObjectID) error
	List(ctx context.Context, filter bson.M, pagination mongo.PaginationOptions) (*mongo.PaginatedResult[Account], error)
	Count(ctx context.Context, filter bson.M) (int64, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	ExistsByUsername(ctx context.Context, username string) (bool, error)
}

type accountRepository struct {
	repo mongo.Repository[Account]
}

func NewAccountRepository(mongoService *mongo.MongoService) AccountRepository {
	return &accountRepository{
		repo: mongo.NewRepository[Account](mongoService, CollectionName),
	}
}

func (r *accountRepository) Create(ctx context.Context, account *Account) (*Account, error) {
	account.ID = primitive.NewObjectID()
	
	result, err := r.repo.Create(ctx, *account)
	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}
	
	return result, nil
}

func (r *accountRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*Account, error) {
	filter := bson.M{"_id": id}
	result, err := r.repo.FindOne(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get account by ID: %w", err)
	}
	
	if result == nil {
		return nil, nil
	}
	
	return result, nil
}

func (r *accountRepository) GetByEmail(ctx context.Context, email string) (*Account, error) {
	filter := bson.M{"email": email}
	result, err := r.repo.FindOne(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get account by email: %w", err)
	}
	
	if result == nil {
		return nil, nil
	}
	
	return result, nil
}

func (r *accountRepository) GetByUsername(ctx context.Context, username string) (*Account, error) {
	filter := bson.M{"username": username}
	result, err := r.repo.FindOne(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get account by username: %w", err)
	}
	
	if result == nil {
		return nil, nil
	}
	
	return result, nil
}

func (r *accountRepository) Update(ctx context.Context, id primitive.ObjectID, updateData bson.M) (*Account, error) {
	filter := bson.M{"_id": id}
	update := bson.M{"$set": updateData}
	
	result, err := r.repo.Update(ctx, filter, update)
	if err != nil {
		return nil, fmt.Errorf("failed to update account: %w", err)
	}
	
	if result == nil {
		return nil, nil
	}
	
	return result, nil
}

func (r *accountRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	filter := bson.M{"_id": id}
	
	err := r.repo.Delete(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}
	
	return nil
}

func (r *accountRepository) List(ctx context.Context, filter bson.M, pagination mongo.PaginationOptions) (*mongo.PaginatedResult[Account], error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	
	result, err := r.repo.FindWithPagination(ctx, filter, pagination, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}
	
	return result, nil
}

func (r *accountRepository) Count(ctx context.Context, filter bson.M) (int64, error) {
	count, err := r.repo.Count(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count accounts: %w", err)
	}
	
	return count, nil
}

func (r *accountRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	filter := bson.M{"email": email}
	count, err := r.repo.Count(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("failed to check if email exists: %w", err)
	}
	
	return count > 0, nil
}

func (r *accountRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	filter := bson.M{"username": username}
	count, err := r.repo.Count(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("failed to check if username exists: %w", err)
	}
	
	return count > 0, nil
}