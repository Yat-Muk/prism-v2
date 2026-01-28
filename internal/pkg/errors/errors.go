package errors

import (
	"errors"
	"fmt"
)

// 預定義錯誤類型
var (
	// 配置相關
	ErrConfigNotFound    = errors.New("configuration file not found")
	ErrConfigInvalid     = errors.New("configuration is invalid")
	ErrConfigParseFailed = errors.New("failed to parse configuration")

	// 協議相關
	ErrProtocolNotFound = errors.New("protocol not found")
	ErrProtocolDisabled = errors.New("protocol is disabled")
	ErrInvalidPort      = errors.New("invalid port number")
	ErrInvalidDomain    = errors.New("invalid domain name")

	// 證書相關
	ErrCertNotFound     = errors.New("certificate not found")
	ErrCertExpired      = errors.New("certificate has expired")
	ErrCertObtainFailed = errors.New("failed to obtain certificate")

	// 系統相關
	ErrServiceNotRunning = errors.New("service is not running")
	ErrPermissionDenied  = errors.New("permission denied")
	ErrCommandFailed     = errors.New("command execution failed")
)

// Error 自定義錯誤類型
type Error struct {
	Code    string
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error {
	return e.Err
}

// New 創建新錯誤
func New(code, message string) error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// Wrap 包裝錯誤
func Wrap(err error, code, message string) error {
	return &Error{
		Code:    code,
		Message: message,
		Err:     err,
	}
}
