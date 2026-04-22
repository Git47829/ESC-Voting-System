package server

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

type pendingVerificationEntry struct {
	code      string
	role      string
	createdAt int64
}

var (
	pendingVerificationMu       sync.Mutex
	pendingVerificationFallback = map[string]pendingVerificationEntry{}
)

func InitRedis() error {
	addr := os.Getenv("REDIS_URL")
	if addr == "" {
		addr = "redis:6379"
	}

	RedisClient = redis.NewClient(&redis.Options{
		Addr:         addr,
		DB:           0,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := RedisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}

	Logger.Info("Redis connection established", slog.String("addr", addr))
	return nil
}

func redisAvailable() bool {
	return RedisClient != nil
}

// CheckRateLimit uses Redis INCR + EXPIRE for distributed rate limiting.
// Returns true if request is allowed.
func CheckRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	if !redisAvailable() {
		return true, nil
	}
	pipe := RedisClient.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, err
	}
	return incr.Val() <= int64(limit), nil
}

// IsTokenUsed checks if a token has already been used via Redis SET.
func IsTokenUsed(ctx context.Context, token string) (bool, error) {
	if !redisAvailable() {
		return false, nil
	}
	return RedisClient.SIsMember(ctx, "used_tokens", token).Result()
}

// MarkTokenUsed adds a token to the used set with TTL.
func MarkTokenUsed(ctx context.Context, token string) error {
	if !redisAvailable() {
		return nil
	}
	pipe := RedisClient.Pipeline()
	pipe.SAdd(ctx, "used_tokens", token)
	pipe.Expire(ctx, "used_tokens", 24*time.Hour)
	_, err := pipe.Exec(ctx)
	return err
}

// StoreToken stores a verification token with expiry in Redis.
func StoreToken(ctx context.Context, token string, ttl time.Duration) error {
	if !redisAvailable() {
		return nil
	}
	return RedisClient.Set(ctx, "token:"+token, "1", ttl).Err()
}

// CheckAndEvictToken atomically checks and deletes a token. Returns true if it existed.
func CheckAndEvictToken(ctx context.Context, token string) (bool, error) {
	if !redisAvailable() {
		return true, nil
	}
	result, err := RedisClient.Del(ctx, "token:"+token).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

// StorePendingVerification stores a 2FA pending verification in Redis.
func StorePendingVerification(ctx context.Context, email, code, role string) error {
	if !redisAvailable() {
		pendingVerificationMu.Lock()
		pendingVerificationFallback[email] = pendingVerificationEntry{
			code:      code,
			role:      role,
			createdAt: time.Now().Unix(),
		}
		pendingVerificationMu.Unlock()
		return nil
	}
	key := "pending_2fa:" + email
	pipe := RedisClient.Pipeline()
	pipe.HSet(ctx, key, map[string]any{
		"code":       code,
		"role":       role,
		"created_at": time.Now().Unix(),
	})
	pipe.Expire(ctx, key, 5*time.Minute)
	_, err := pipe.Exec(ctx)
	return err
}

// GetAndDeletePendingVerification retrieves and deletes a pending verification.
// Returns code, role, createdAt, exists.
func GetAndDeletePendingVerification(ctx context.Context, email string) (string, string, int64, bool, error) {
	if !redisAvailable() {
		pendingVerificationMu.Lock()
		entry, ok := pendingVerificationFallback[email]
		if ok {
			delete(pendingVerificationFallback, email)
		}
		pendingVerificationMu.Unlock()
		if !ok {
			return "", "", 0, false, nil
		}
		if time.Since(time.Unix(entry.createdAt, 0)) > 5*time.Minute {
			return "", "", 0, false, nil
		}
		return entry.code, entry.role, entry.createdAt, true, nil
	}
	key := "pending_2fa:" + email
	result, err := RedisClient.HGetAll(ctx, key).Result()
	if err != nil {
		return "", "", 0, false, err
	}
	if len(result) == 0 {
		return "", "", 0, false, nil
	}

	code := result["code"]
	role := result["role"]
	var createdAt int64
	fmt.Sscanf(result["created_at"], "%d", &createdAt)

	RedisClient.Del(ctx, key)
	return code, role, createdAt, true, nil
}

// CacheSet stores a value in Redis with TTL.
func CacheSet(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if !redisAvailable() {
		return nil
	}
	return RedisClient.Set(ctx, "cache:"+key, value, ttl).Err()
}

// CacheGet retrieves a cached value. Returns nil, nil on cache miss.
func CacheGet(ctx context.Context, key string) ([]byte, error) {
	if !redisAvailable() {
		return nil, nil
	}
	val, err := RedisClient.Get(ctx, "cache:"+key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	return val, err
}

// CacheInvalidate deletes cache keys matching a pattern.
func CacheInvalidate(ctx context.Context, patterns ...string) {
	if !redisAvailable() {
		return
	}
	for _, p := range patterns {
		iter := RedisClient.Scan(ctx, 0, "cache:"+p, 100).Iterator()
		for iter.Next(ctx) {
			RedisClient.Del(ctx, iter.Val())
		}
	}
}
