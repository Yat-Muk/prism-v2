package appctx

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestContext 測試上下文功能
func TestContext(t *testing.T) {
	t.Run("創建上下文", func(t *testing.T) {
		ctx := New()
		assert.NotNil(t, ctx)
	})

	t.Run("帶值的上下文", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithValue(ctx, "key", "value")

		val := ctx.Value("key")
		assert.Equal(t, "value", val)
	})
}

// TestRequestID 測試請求ID
func TestRequestID(t *testing.T) {
	ctx := context.Background()

	t.Run("設置和獲取請求ID", func(t *testing.T) {
		ctx = WithRequestID(ctx, "req-123")
		id := GetRequestID(ctx)
		assert.Equal(t, "req-123", id)
	})

	t.Run("無請求ID返回空", func(t *testing.T) {
		ctx := context.Background()
		id := GetRequestID(ctx)
		assert.Empty(t, id)
	})
}

// TestUserContext 測試用戶上下文
func TestUserContext(t *testing.T) {
	ctx := context.Background()

	t.Run("設置和獲取用戶", func(t *testing.T) {
		ctx = WithUser(ctx, "admin")
		user := GetUser(ctx)
		assert.Equal(t, "admin", user)
	})
}

// 輔助函數
func New() context.Context {
	return context.Background()
}

func WithValue(ctx context.Context, key, val interface{}) context.Context {
	return context.WithValue(ctx, key, val)
}

type contextKey string

const (
	requestIDKey contextKey = "requestID"
	userKey      contextKey = "user"
)

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

func WithUser(ctx context.Context, user string) context.Context {
	return context.WithValue(ctx, userKey, user)
}

func GetUser(ctx context.Context) string {
	if user, ok := ctx.Value(userKey).(string); ok {
		return user
	}
	return ""
}
