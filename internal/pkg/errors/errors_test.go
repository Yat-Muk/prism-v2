package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestWrapFunction 測試Wrap函數
func TestWrapFunction(t *testing.T) {
	baseErr := errors.New("base error")

	t.Run("Wrap返回非nil", func(t *testing.T) {
		wrapped := Wrap(baseErr, "TestError", "context")
		assert.Error(t, wrapped)
		assert.NotNil(t, wrapped)
	})

	t.Run("Wrap保留原錯誤", func(t *testing.T) {
		wrapped := Wrap(baseErr, "TestError", "context")
		assert.True(t, errors.Is(wrapped, baseErr))
	})

	t.Run("Wrap nil創建新錯誤", func(t *testing.T) {
		// Wrap(nil) 會創建一個新的錯誤，而不是返回nil
		wrapped := Wrap(nil, "TestError", "context")
		assert.Error(t, wrapped) // 應該返回錯誤
		assert.NotNil(t, wrapped)
		assert.Contains(t, wrapped.Error(), "TestError")
	})
}

// TestNewFunction 測試New函數（2個參數版本）
func TestNewFunction(t *testing.T) {
	t.Run("創建錯誤", func(t *testing.T) {
		err := New("TestError", "test error message")
		assert.Error(t, err)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "test error")
	})

	t.Run("錯誤消息非空", func(t *testing.T) {
		err := New("Type1", "message1")
		assert.NotEmpty(t, err.Error())
	})

	t.Run("不同類型的錯誤", func(t *testing.T) {
		err1 := New("Type1", "message1")
		err2 := New("Type2", "message2")

		assert.NotEqual(t, err1.Error(), err2.Error())
	})
}

// TestErrorMessages 測試錯誤消息
func TestErrorMessages(t *testing.T) {
	t.Run("自定義錯誤", func(t *testing.T) {
		err := New("CustomError", "test error")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "test")
	})

	t.Run("包裝錯誤消息", func(t *testing.T) {
		baseErr := New("BaseError", "base message")
		wrapped := Wrap(baseErr, "WrapperError", "additional context")

		msg := wrapped.Error()
		assert.NotEmpty(t, msg)
		assert.Contains(t, msg, "context")
	})
}

// TestErrorTypes 測試錯誤類型
func TestErrorTypes(t *testing.T) {
	t.Run("New函數創建錯誤", func(t *testing.T) {
		err := New("TestType", "test error")
		assert.Error(t, err)
	})

	t.Run("Is函數判斷", func(t *testing.T) {
		err1 := New("Error1", "message1")
		err2 := Wrap(err1, "Error2", "wrapped")

		assert.True(t, errors.Is(err2, err1))
	})
}

// TestErrorComparison 測試錯誤比較
func TestErrorComparison(t *testing.T) {
	err1 := New("Type1", "error1")
	err2 := New("Type2", "error2")

	assert.NotEqual(t, err1.Error(), err2.Error())

	// 相同的錯誤
	err3 := New("Type1", "error1")
	// 注意：兩次調用New即使參數相同，也會創建不同的錯誤對象
	// 所以比較錯誤消息而不是對象本身
	assert.Equal(t, err1.Error(), err3.Error())
}

// TestErrorWrapping 測試錯誤包裝
func TestErrorWrapping(t *testing.T) {
	t.Run("包裝標準錯誤", func(t *testing.T) {
		baseErr := errors.New("standard error")
		wrapped := Wrap(baseErr, "CustomType", "context info")

		assert.Error(t, wrapped)
		assert.True(t, errors.Is(wrapped, baseErr))
	})

	t.Run("多層包裝", func(t *testing.T) {
		err1 := New("Level1", "base error")
		err2 := Wrap(err1, "Level2", "context 2")
		err3 := Wrap(err2, "Level3", "context 3")

		assert.True(t, errors.Is(err3, err1))
		assert.True(t, errors.Is(err3, err2))
	})
}

// TestErrorWithContext 測試帶上下文的錯誤
func TestErrorWithContext(t *testing.T) {
	t.Run("添加上下文信息", func(t *testing.T) {
		baseErr := New("DatabaseError", "connection failed")
		withContext := Wrap(baseErr, "OperationError", "failed to connect to DB")

		assert.Contains(t, withContext.Error(), "failed to connect")
	})
}

// TestWrapNilBehavior 測試Wrap處理nil的行為
func TestWrapNilBehavior(t *testing.T) {
	t.Run("Wrap nil創建新錯誤而非返回nil", func(t *testing.T) {
		result := Wrap(nil, "SomeType", "context")
		// 實際行為：Wrap(nil) 創建一個新錯誤
		assert.Error(t, result)
		assert.NotNil(t, result)
		assert.Contains(t, result.Error(), "SomeType")
		assert.Contains(t, result.Error(), "context")
	})
}

// TestErrorFormatting 測試錯誤格式
func TestErrorFormatting(t *testing.T) {
	t.Run("錯誤包含類型和消息", func(t *testing.T) {
		err := New("ValidationError", "field is required")
		errMsg := err.Error()

		assert.Contains(t, errMsg, "ValidationError")
		assert.Contains(t, errMsg, "field is required")
	})

	t.Run("包裝錯誤包含上下文", func(t *testing.T) {
		base := New("DBError", "connection timeout")
		wrapped := Wrap(base, "ServiceError", "failed to query database")

		errMsg := wrapped.Error()
		assert.Contains(t, errMsg, "ServiceError")
		assert.Contains(t, errMsg, "failed to query")
	})
}
