package account

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/redis"
)

type CacheSessionRepository struct {
	cacheService redis.RedisService
}

var _ SessionRepository = (*CacheSessionRepository)(nil)

func NewCacheSessionRepository(cacheService redis.RedisService) *CacheSessionRepository {
	return &CacheSessionRepository{
		cacheService: cacheService,
	}
}

func (r *CacheSessionRepository) sessionKey(tokenHash string) string {
	return fmt.Sprintf("session:%s", tokenHash)
}

func (r *CacheSessionRepository) userSessionsKey(accountID string) string {
	return fmt.Sprintf("user_sessions:%s", accountID)
}

func (r *CacheSessionRepository) sessionLastUsedKey(tokenHash string) string {
	return fmt.Sprintf("session_last_used:%s", tokenHash)
}

func (r *CacheSessionRepository) CreateSession(ctx context.Context, session *Session) (*Session, error) {
	if session.ID == primitive.NilObjectID {
		session.ID = primitive.NewObjectID()
	}

	now := time.Now()
	session.CreatedAt = now
	session.LastUsedAt = now

	sessionData, err := json.Marshal(session)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal session data: %w", err)
	}

	var ttl time.Duration
	if !session.ExpiresAt.IsZero() {
		ttl = time.Until(session.ExpiresAt)
		if ttl <= 0 {
			return nil, fmt.Errorf("session expiration time is in the past")
		}
	} else {
		ttl = 24 * time.Hour
		session.ExpiresAt = now.Add(ttl)
	}

	sessionKey := r.sessionKey(session.TokenHash)
	if err := r.cacheService.Set(ctx, sessionKey, sessionData, ttl); err != nil {
		return nil, fmt.Errorf("failed to store session data: %w", err)
	}

	userSessionsKey := r.userSessionsKey(session.AccountID)
	if _, err := r.cacheService.SAdd(ctx, userSessionsKey, session.TokenHash); err != nil {
		return nil, fmt.Errorf("failed to add session to user sessions: %w", err)
	}

	if err := r.cacheService.Expire(ctx, userSessionsKey, ttl+time.Hour); err != nil {
		return nil, fmt.Errorf("failed to set expiration for user sessions: %w", err)
	}

	lastUsedKey := r.sessionLastUsedKey(session.TokenHash)
	if err := r.cacheService.Set(ctx, lastUsedKey, strconv.FormatInt(now.Unix(), 10), ttl); err != nil {
		return nil, fmt.Errorf("failed to set last used timestamp: %w", err)
	}

	return session, nil
}

func (r *CacheSessionRepository) GetSessionByToken(ctx context.Context, tokenHash string) (*Session, error) {
	sessionKey := r.sessionKey(tokenHash)

	sessionDataStr, err := r.cacheService.Get(ctx, sessionKey)
	if err != nil {
		if exists, existsErr := r.cacheService.Exists(ctx, sessionKey); existsErr == nil && exists == 0 {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get session data: %w", err)
	}

	var session Session
	if err := json.Unmarshal([]byte(sessionDataStr), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session data: %w", err)
	}

	lastUsedKey := r.sessionLastUsedKey(tokenHash)
	if lastUsedStr, err := r.cacheService.Get(ctx, lastUsedKey); err == nil {
		if lastUsedUnix, parseErr := strconv.ParseInt(lastUsedStr, 10, 64); parseErr == nil {
			session.LastUsedAt = time.Unix(lastUsedUnix, 0)
		}
	}

	if session.IsExpired() {
		r.DeactivateSession(ctx, tokenHash)
		return nil, nil
	}

	return &session, nil
}

func (r *CacheSessionRepository) GetSessionsByAccountID(ctx context.Context, accountID string) ([]*Session, error) {
	userSessionsKey := r.userSessionsKey(accountID)

	tokenHashes, err := r.cacheService.SMembers(ctx, userSessionsKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
	}

	if len(tokenHashes) == 0 {
		return []*Session{}, nil
	}

	sessions := make([]*Session, 0, len(tokenHashes))

	for _, tokenHash := range tokenHashes {
		session, err := r.GetSessionByToken(ctx, tokenHash)
		if err != nil {
			continue
		}

		if session != nil {
			sessions = append(sessions, session)
		} else {
			r.cacheService.SRemove(ctx, userSessionsKey, tokenHash)
		}
	}

	return sessions, nil
}

func (r *CacheSessionRepository) UpdateSessionLastUsed(ctx context.Context, id primitive.ObjectID) error {
	return fmt.Errorf("UpdateSessionLastUsed by ID not supported in cache implementation - use UpdateSessionLastUsedByToken")
}

func (r *CacheSessionRepository) UpdateSessionLastUsedByToken(ctx context.Context, tokenHash string) error {
	sessionKey := r.sessionKey(tokenHash)

	exists, err := r.cacheService.Exists(ctx, sessionKey)
	if err != nil {
		return fmt.Errorf("failed to check session existence: %w", err)
	}

	if exists == 0 {
		return fmt.Errorf("session not found or expired")
	}

	now := time.Now()

	lastUsedKey := r.sessionLastUsedKey(tokenHash)
	if err := r.cacheService.Set(ctx, lastUsedKey, strconv.FormatInt(now.Unix(), 10), time.Hour); err != nil {
		return fmt.Errorf("failed to update last used timestamp: %w", err)
	}

	currentTTL, err := r.cacheService.TTL(ctx, sessionKey)
	if err != nil {
		return fmt.Errorf("failed to get session TTL: %w", err)
	}

	if currentTTL < time.Hour {
		if err := r.cacheService.Expire(ctx, sessionKey, 24*time.Hour); err != nil {
			return fmt.Errorf("failed to extend session expiration: %w", err)
		}

		if err := r.cacheService.Expire(ctx, lastUsedKey, 24*time.Hour); err != nil {
			return fmt.Errorf("failed to extend last used key expiration: %w", err)
		}
	}

	return nil
}

func (r *CacheSessionRepository) DeactivateSession(ctx context.Context, tokenHash string) error {
	sessionKey := r.sessionKey(tokenHash)
	lastUsedKey := r.sessionLastUsedKey(tokenHash)

	sessionDataStr, err := r.cacheService.Get(ctx, sessionKey)
	var accountID string
	if err == nil {
		var session Session
		if json.Unmarshal([]byte(sessionDataStr), &session) == nil {
			accountID = session.AccountID
		}
	}

	keys := []string{sessionKey, lastUsedKey}
	if _, err := r.cacheService.Delete(ctx, keys...); err != nil {
		return fmt.Errorf("failed to delete session keys: %w", err)
	}

	if accountID != "" {
		userSessionsKey := r.userSessionsKey(accountID)
		if _, err := r.cacheService.SRemove(ctx, userSessionsKey, tokenHash); err != nil {
			return fmt.Errorf("failed to remove session from user sessions: %w", err)
		}
	}

	return nil
}

func (r *CacheSessionRepository) DeactivateAllUserSessions(ctx context.Context, accountID string) error {
	userSessionsKey := r.userSessionsKey(accountID)

	tokenHashes, err := r.cacheService.SMembers(ctx, userSessionsKey)
	if err != nil {
		return fmt.Errorf("failed to get user sessions for deactivation: %w", err)
	}

	if len(tokenHashes) == 0 {
		return nil
	}

	for _, tokenHash := range tokenHashes {
		if err := r.DeactivateSession(ctx, tokenHash); err != nil {
			return fmt.Errorf("failed to deactivate session %s: %w", tokenHash, err)
		}
	}

	if _, err := r.cacheService.Delete(ctx, userSessionsKey); err != nil {
		return fmt.Errorf("failed to clear user sessions set: %w", err)
	}

	return nil
}

func (r *CacheSessionRepository) CleanupExpiredSessions(ctx context.Context) error {
	userSessionsPattern := "user_sessions:*"
	userSessionsKeys, err := r.cacheService.Keys(ctx, userSessionsPattern)
	if err != nil {
		return fmt.Errorf("failed to get user sessions keys: %w", err)
	}

	for _, userSessionsKey := range userSessionsKeys {
		tokenHashes, err := r.cacheService.SMembers(ctx, userSessionsKey)
		if err != nil {
			continue
		}

		for _, tokenHash := range tokenHashes {
			sessionKey := r.sessionKey(tokenHash)
			exists, err := r.cacheService.Exists(ctx, sessionKey)
			if err != nil || exists == 0 {
				r.cacheService.SRemove(ctx, userSessionsKey, tokenHash)
			}
		}
	}

	return nil
}
