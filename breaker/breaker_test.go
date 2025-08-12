package breaker

import (
	"errors"
	"testing"
	"time"
)

func TestNewBreaker(t *testing.T) {
	settings := Settings{
		Name:        "test",
		MaxRequests: 10,
		Interval:    5 * time.Second,
		Timeout:     30 * time.Second,
	}

	b := NewBreaker(settings)

	if b.Name() != "test" {
		t.Errorf("expected name 'test', got '%s'", b.Name())
	}

	if b.State() != Closed {
		t.Errorf("expected initial state Closed, got %v", b.State())
	}

	counts := b.Counts()
	if counts.Requests != 0 {
		t.Errorf("expected 0 requests, got %d", counts.Requests)
	}
}

func TestBreakerDefaultSettings(t *testing.T) {
	settings := Settings{
		Name: "test",
	}

	b := NewBreaker(settings)

	if b.maxRequests != 1 {
		t.Errorf("expected maxRequests 1, got %d", b.maxRequests)
	}

	if b.interval != defaultInterval {
		t.Errorf("expected interval %v, got %v", defaultInterval, b.interval)
	}

	if b.timeout != defaultTimeout {
		t.Errorf("expected timeout %v, got %v", defaultTimeout, b.timeout)
	}
}

func TestBreakerExecuteSuccess(t *testing.T) {
	b := NewBreaker(Settings{Name: "test"})

	result, err := b.Execute(func() (any, error) {
		return "success", nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if result != "success" {
		t.Errorf("expected result 'success', got %v", result)
	}

	counts := b.Counts()
	if counts.TotalSuccesses != 1 {
		t.Errorf("expected 1 success, got %d", counts.TotalSuccesses)
	}

	if counts.ConsecutiveSuccesses != 1 {
		t.Errorf("expected 1 consecutive success, got %d", counts.ConsecutiveSuccesses)
	}
}

func TestBreakerExecuteFailure(t *testing.T) {
	b := NewBreaker(Settings{Name: "test"})
	testErr := errors.New("test error")

	result, err := b.Execute(func() (any, error) {
		return nil, testErr
	})

	if err != testErr {
		t.Errorf("expected error %v, got %v", testErr, err)
	}

	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}

	counts := b.Counts()
	if counts.TotalFailures != 1 {
		t.Errorf("expected 1 failure, got %d", counts.TotalFailures)
	}

	if counts.ConsecutiveFailures != 1 {
		t.Errorf("expected 1 consecutive failure, got %d", counts.ConsecutiveFailures)
	}
}

func TestBreakerExecutePanic(t *testing.T) {
	b := NewBreaker(Settings{Name: "test"})

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic")
		}
	}()

	b.Execute(func() (any, error) {
		panic("test panic")
	})
}

func TestBreakerStateTransition(t *testing.T) {
	// 创建断路器，设置较少的失败次数阈值
	settings := Settings{
		Name:        "test",
		MaxRequests: 1,
		Interval:    1 * time.Millisecond,
		Timeout:     1 * time.Millisecond,
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures >= 2
		},
	}

	b := NewBreaker(settings)

	// 第一次失败
	b.Execute(func() (any, error) {
		return nil, errors.New("error 1")
	})

	if b.State() != Closed {
		t.Errorf("expected state Closed after 1 failure, got %v", b.State())
	}

	// 第二次失败，应该触发断路
	b.Execute(func() (any, error) {
		return nil, errors.New("error 2")
	})

	// 检查状态应该是Open
	if b.State() != Open {
		t.Errorf("expected state Open after 2 failures, got %v", b.State())
	}

	// 在Open状态下执行应该失败
	_, err := b.Execute(func() (any, error) {
		return "success", nil
	})

	if err != ErrOpenState {
		t.Errorf("expected error %v, got %v", ErrOpenState, err)
	}
}

func TestBreakerHalfOpenState(t *testing.T) {
	settings := Settings{
		Name:        "test",
		MaxRequests: 2,
		Interval:    1 * time.Millisecond,
		Timeout:     1 * time.Millisecond,
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures >= 1
		},
	}

	b := NewBreaker(settings)

	// 触发断路
	b.Execute(func() (any, error) {
		return nil, errors.New("error")
	})

	time.Sleep(2 * time.Millisecond)

	if b.State() != HalfOpen {
		t.Errorf("expected state HalfOpen, got %v", b.State())
	}

	// 在HalfOpen状态下成功执行
	b.Execute(func() (any, error) {
		return "success", nil
	})

	// 再次成功执行，应该回到Closed状态
	b.Execute(func() (any, error) {
		return "success", nil
	})

	if b.State() != Closed {
		t.Errorf("expected state Closed after 2 successes, got %v", b.State())
	}
}

func TestBreakerHalfOpenTooManyRequests(t *testing.T) {
	settings := Settings{
		Name:        "test",
		MaxRequests: 1,
		Interval:    1 * time.Millisecond,
		Timeout:     1 * time.Millisecond,
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures >= 1
		},
	}

	b := NewBreaker(settings)

	// 触发断路
	b.Execute(func() (any, error) {
		return nil, errors.New("error")
	})

	time.Sleep(2 * time.Millisecond)

	if b.State() != HalfOpen {
		t.Errorf("expected state HalfOpen, got %v", b.State())
	}

	// 第一次请求应该成功
	_, err := b.Execute(func() (any, error) {
		return "success", nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// 在HalfOpen状态下，第一次成功请求后应该回到Closed状态
	if b.State() != Closed {
		t.Errorf("expected state Closed after success, got %v", b.State())
	}

	// 现在在Closed状态下，应该可以正常处理请求
	_, err = b.Execute(func() (any, error) {
		return "success", nil
	})

	if err != nil {
		t.Errorf("expected no error in Closed state, got %v", err)
	}
}

func TestBreakerCustomReadyToTrip(t *testing.T) {
	customReadyToTrip := func(counts Counts) bool {
		return counts.TotalFailures >= 3
	}

	settings := Settings{
		Name:        "test",
		ReadyToTrip: customReadyToTrip,
	}

	b := NewBreaker(settings)

	// 前两次失败不应该触发断路
	for i := 0; i < 2; i++ {
		b.Execute(func() (any, error) {
			return nil, errors.New("error")
		})
	}

	if b.State() != Closed {
		t.Errorf("expected state Closed after 2 failures, got %v", b.State())
	}

	// 第三次失败应该触发断路
	b.Execute(func() (any, error) {
		return nil, errors.New("error")
	})

	if b.State() != Open {
		t.Errorf("expected state Open after 3 failures, got %v", b.State())
	}
}

func TestBreakerCustomIsSuccessful(t *testing.T) {
	customIsSuccessful := func(err error) bool {
		return err == nil || err.Error() == "acceptable error"
	}

	settings := Settings{
		Name:         "test",
		IsSuccessful: customIsSuccessful,
	}

	b := NewBreaker(settings)

	// 可接受的错误应该被认为是成功的
	b.Execute(func() (any, error) {
		return nil, errors.New("acceptable error")
	})

	counts := b.Counts()
	if counts.TotalSuccesses != 1 {
		t.Errorf("expected 1 success, got %d", counts.TotalSuccesses)
	}

	// 不可接受的错误应该被认为是失败的
	b.Execute(func() (any, error) {
		return nil, errors.New("unacceptable error")
	})

	counts = b.Counts()
	if counts.TotalFailures != 1 {
		t.Errorf("expected 1 failure, got %d", counts.TotalFailures)
	}
}

func TestBreakerStateChangeCallback(t *testing.T) {
	var stateChanges []State
	var stateChangeNames []string

	onStateChange := func(name string, from, to State) {
		stateChanges = append(stateChanges, to)
		stateChangeNames = append(stateChangeNames, name)
	}

	settings := Settings{
		Name:          "test",
		OnStateChange: onStateChange,
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures >= 1
		},
	}

	b := NewBreaker(settings)

	// 触发状态变化
	b.Execute(func() (any, error) {
		return nil, errors.New("error")
	})

	if len(stateChanges) == 0 {
		t.Error("expected state change callback to be called")
	}

	if stateChangeNames[0] != "test" {
		t.Errorf("expected name 'test', got '%s'", stateChangeNames[0])
	}
}

func TestTwoStepBreaker(t *testing.T) {
	settings := Settings{
		Name:        "test",
		MaxRequests: 1,
		Interval:    1 * time.Millisecond,
		Timeout:     1 * time.Millisecond,
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures >= 1
		},
	}

	tb := NewTwoStepBreaker(settings)

	if tb.Name() != "test" {
		t.Errorf("expected name 'test', got '%s'", tb.Name())
	}

	if tb.State() != Closed {
		t.Errorf("expected initial state Closed, got %v", tb.State())
	}

	// 测试Allow方法
	done, err := tb.Allow()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if done == nil {
		t.Error("expected done function")
	}

	// 标记为成功
	done(true)

	counts := tb.Counts()
	if counts.TotalSuccesses != 1 {
		t.Errorf("expected 1 success, got %d", counts.TotalSuccesses)
	}
}

func TestBreakerGeneration(t *testing.T) {
	settings := Settings{
		Name:     "test",
		Interval: 1 * time.Millisecond,
		Timeout:  1 * time.Millisecond,
	}

	b := NewBreaker(settings)

	initialGen := b.generation

	// 等待间隔时间，应该生成新的generation
	time.Sleep(2 * time.Millisecond)

	// 触发状态检查
	b.State()

	if b.generation <= initialGen {
		t.Error("expected generation to increase")
	}
}

func TestBreakerCountsClear(t *testing.T) {
	settings := Settings{
		Name:     "test",
		Interval: 1 * time.Millisecond,
	}
	b := NewBreaker(settings)

	// 添加一些计数
	b.Execute(func() (any, error) {
		return "success", nil
	})

	b.Execute(func() (any, error) {
		return nil, errors.New("error")
	})

	counts := b.Counts()
	if counts.Requests != 2 {
		t.Errorf("expected 2 requests, got %d", counts.Requests)
	}

	// 等待间隔时间，计数应该被清除
	time.Sleep(2 * time.Millisecond)
	b.State()

	counts = b.Counts()
	if counts.Requests != 0 {
		t.Errorf("expected 0 requests after clear, got %d", counts.Requests)
	}
}
