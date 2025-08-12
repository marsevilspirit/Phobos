package errors

import (
	"fmt"
	"time"
)

// ErrorCode 定义错误代码类型
type ErrorCode int

// 错误代码常量
const (
	ErrCodeSuccess ErrorCode = iota
	ErrCodeInvalidRequest
	ErrCodeServiceUnavailable
	ErrCodeTimeout
	ErrCodeInternalError
	ErrCodeUnauthorized
	ErrCodeForbidden
	ErrCodeNotFound
	ErrCodeValidationFailed
	ErrCodeRateLimitExceeded
)

// Error 增强的错误结构
type Error struct {
	Code      ErrorCode
	Message   string
	Details   map[string]interface{}
	Timestamp time.Time
	Cause     error
}

// New 创建新的错误
func New(code ErrorCode, message string) *Error {
	return &Error{
		Code:      code,
		Message:   message,
		Details:   make(map[string]interface{}),
		Timestamp: time.Now(),
	}
}

// WithDetail 添加错误详情
func (e *Error) WithDetail(key string, value interface{}) *Error {
	e.Details[key] = value
	return e
}

// WithCause 设置错误原因
func (e *Error) WithCause(cause error) *Error {
	e.Cause = cause
	return e
}

// Error 实现error接口
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 返回错误原因
func (e *Error) Unwrap() error {
	return e.Cause
}

// Is 检查错误代码是否匹配
func (e *Error) Is(target error) bool {
	if target, ok := target.(*Error); ok {
		return e.Code == target.Code
	}
	return false
}

// MultiError 多错误集合
type MultiError struct {
	Errors []error
	Code   ErrorCode
}

// NewMultiError 创建多错误
func NewMultiError(errors []error) *MultiError {
	return &MultiError{
		Errors: errors,
		Code:   ErrCodeInternalError,
	}
}

// Add 添加错误
func (e *MultiError) Add(err error) {
	e.Errors = append(e.Errors, err)
}

// Error 实现error接口
func (e *MultiError) Error() string {
	if len(e.Errors) == 0 {
		return "no errors"
	}
	return fmt.Sprintf("%d errors: %v", len(e.Errors), e.Errors)
}

// HasErrors 检查是否有错误
func (e *MultiError) HasErrors() bool {
	return len(e.Errors) > 0
}

// 预定义错误
var (
	ErrInvalidRequest     = New(ErrCodeInvalidRequest, "invalid request")
	ErrServiceUnavailable = New(ErrCodeServiceUnavailable, "service unavailable")
	ErrTimeout           = New(ErrCodeTimeout, "request timeout")
	ErrInternalError     = New(ErrCodeInternalError, "internal error")
	ErrUnauthorized      = New(ErrCodeUnauthorized, "unauthorized")
	ErrForbidden         = New(ErrCodeForbidden, "forbidden")
	ErrNotFound          = New(ErrCodeNotFound, "not found")
)
