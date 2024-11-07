package breaker

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStateConstants(t *testing.T) {
	assert.Equal(t, State(0), Closed)
	assert.Equal(t, State(1), HalfOpen)
	assert.Equal(t, State(2), Open)

	assert.Equal(t, Closed.String(), "closed")
	assert.Equal(t, HalfOpen.String(), "half-open")
	assert.Equal(t, Open.String(), "open")
	assert.Equal(t, State(100).String(), "unknown state")
}

func TestCounts(t *testing.T) {
	c := &Counts{}

	c.onRequest()
	assert.Equal(t, uint32(1), c.Requests)

	c.onSuccess()
	assert.Equal(t, uint32(1), c.TotalSuccesses)
	assert.Equal(t, uint32(1), c.ConsecutiveSuccesses)
	assert.Equal(t, uint32(0), c.ConsecutiveFailures)

	c.onFailure()
	assert.Equal(t, uint32(1), c.TotalFailures)
	assert.Equal(t, uint32(1), c.ConsecutiveFailures)
	assert.Equal(t, uint32(0), c.ConsecutiveSuccesses)

	c.clear()
	assert.Equal(t, uint32(0), c.Requests)
	assert.Equal(t, uint32(0), c.TotalSuccesses)
	assert.Equal(t, uint32(0), c.TotalFailures)
	assert.Equal(t, uint32(0), c.ConsecutiveSuccesses)
	assert.Equal(t, uint32(0), c.ConsecutiveFailures)
}

func TestBreaker(t *testing.T) {
	cb := NewBreaker(Settings{
		Name: "test-breaker",
	})

	assert.Equal(t, cb.Name(), "test-breaker")
	assert.Equal(t, cb.State(), Closed)

	// 测试请求和响应处理
	result, err := cb.Execute(func() (interface{}, error) {
		return 42, nil
	})
	assert.Nil(t, err)
	assert.Equal(t, 42, result)
	assert.Equal(t, uint32(1), cb.Counts().Requests)
	assert.Equal(t, uint32(1), cb.Counts().TotalSuccesses)

	// 测试故障处理
	cb.Execute(func() (interface{}, error) {
		return 0, errors.New("error")
	})
	assert.Equal(t, uint32(1), cb.Counts().TotalFailures)
}

func TestTwoStepBreaker(t *testing.T) {
	tb := NewTwoStepBreaker(Settings{
		Name: "test-two-step-breaker",
	})

	assert.Equal(t, tb.Name(), "test-two-step-breaker")
	assert.Equal(t, tb.State(), Closed)

	done, err := tb.Allow()
	assert.Nil(t, err)
	done(true)
	assert.Equal(t, uint32(1), tb.Counts().Requests)
	assert.Equal(t, uint32(1), tb.Counts().TotalSuccesses)

	done, err = tb.Allow()
	assert.Nil(t, err)
	done(false)
	assert.Equal(t, uint32(1), tb.Counts().TotalFailures)
}

func TestCircuitBreakerTripAndReset(t *testing.T) {
	// 创建一个Breaker实例
	b := NewBreaker(Settings{
		Name: "test-breaker",
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures > 2 // 设置阈值为连续失败2次
		},
		Timeout: 1 * time.Second,
	})

	assert.Equal(t, b.State(), Closed)

	// 模拟3次失败的请求，检测熔断状态
	for i := 0; i < 3; i++ {
		result, err := b.Execute(func() (interface{}, error) {
			return 0, errors.New("RPC error")
		})
		assert.NotNil(t, err)
		assert.Equal(t, 0, result) // 修改这个断言，期望返回值是0
	}

	// 检查Breaker是否进入Open状态
	assert.Equal(t, b.State(), Open)

	// 尝试另一个请求，应立即被拒绝
	result, err := b.Execute(func() (interface{}, error) {
		return 1, nil
	})
	assert.Equal(t, err, ErrOpenState)
	assert.Nil(t, result)

	// 模拟复位超时（time.Sleep），并进入Half-Open状态
	time.Sleep(b.timeout) // 等待超时结束

	// 检查Breaker是否进入Half-Open状态
	assert.Equal(t, b.State(), HalfOpen)

	// 模拟成功的请求，应该重新进入Closed状态
	result, err = b.Execute(func() (interface{}, error) {
		return 1, nil
	})
	assert.Nil(t, err)
	assert.Equal(t, 1, result)

	// 检查Breaker是否返回Closed状态
	assert.Equal(t, b.State(), Closed)
}

func TestTwoStepCircuitBreakerTripAndReset(t *testing.T) {
	// 创建一个TwoStepBreaker实例
	tb := NewTwoStepBreaker(Settings{
		Name: "test-two-step-breaker",
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures > 2 // 设置阈值为连续失败2次
		},
		Timeout: 1 * time.Second,
	})

	assert.Equal(t, tb.State(), Closed)

	// 模拟3次失败的请求，检测熔断状态
	for i := 0; i < 3; i++ {
		done, err := tb.Allow()
		assert.Nil(t, err)
		done(false) // 模拟失败
	}

	// 检查TwoStepBreaker是否进入Open状态
	assert.Equal(t, tb.State(), Open)

	// 尝试另一个请求，应立即被拒绝
	done, err := tb.Allow()
	assert.Equal(t, err, ErrOpenState)
	assert.Nil(t, done)

	// 模拟复位超时（time.Sleep），并进入Half-Open状态
	time.Sleep(tb.b.timeout) // 等待超时结束

	// 检查TwoStepBreaker是否进入Half-Open状态
	assert.Equal(t, tb.State(), HalfOpen)

	// 模拟成功的请求，应该重新进入Closed状态
	done, err = tb.Allow()
	assert.Nil(t, err)
	done(true) // 模拟成功

	// 检查TwoStepBreaker是否返回Closed状态
	assert.Equal(t, tb.State(), Closed)
}
