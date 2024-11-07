package breaker

import (
	"errors"
	"sync"
	"time"
)

// State 定义电路断路器的三种状态
type State int

const (
	Closed State = iota
	HalfOpen
	Open
)

const defaultInterval = time.Duration(0) * time.Second
const defaultTimeout = time.Duration(60) * time.Second

var (
	ErrTooManyRequests = errors.New("too many requests")
	ErrOpenState       = errors.New("circuit breaker is open")
)

// String 返回状态的字符串表示
func (s State) String() string {
	switch s {
	case Closed:
		return "closed"
	case HalfOpen:
		return "half-open"
	case Open:
		return "open"
	default:
		return "unknown state"
	}
}

// Counts 记录请求的统计数据
type Counts struct {
	Requests             uint32
	TotalSuccesses       uint32
	TotalFailures        uint32
	ConsecutiveSuccesses uint32
	ConsecutiveFailures  uint32
}

func (c *Counts) onRequest() {
	c.Requests++
}

func (c *Counts) onSuccess() {
	c.TotalSuccesses++
	c.ConsecutiveSuccesses++
	c.ConsecutiveFailures = 0
}

func (c *Counts) onFailure() {
	c.TotalFailures++
	c.ConsecutiveFailures++
	c.ConsecutiveSuccesses = 0
}

func (c *Counts) clear() {
	c.Requests = 0
	c.TotalSuccesses = 0
	c.TotalFailures = 0
	c.ConsecutiveSuccesses = 0
	c.ConsecutiveFailures = 0
}

// Settings 定义电路断路器的配置
type Settings struct {
	Name          string
	MaxRequests   uint32
	Interval      time.Duration
	Timeout       time.Duration
	ReadyToTrip   func(counts Counts) bool
	OnStateChange func(name string, from State, to State)
	IsSuccessful  func(err error) bool
}

// Breaker 定义电路断路器的主体结构
type Breaker struct {
	name          string
	maxRequests   uint32
	interval      time.Duration
	timeout       time.Duration
	readyToTrip   func(counts Counts) bool
	isSuccessful  func(err error) bool
	onStateChange func(name string, from State, to State)

	mu         sync.Mutex
	state      State
	generation uint64
	counts     Counts
	expiry     time.Time
}

// NewBreaker 创建一个新的电路断路器
func NewBreaker(st Settings) *Breaker {
	b := new(Breaker)
	b.name = st.Name
	b.onStateChange = st.OnStateChange

	if st.MaxRequests == 0 {
		b.maxRequests = 1
	} else {
		b.maxRequests = st.MaxRequests
	}

	if st.Interval <= 0 {
		b.interval = defaultInterval
	} else {
		b.interval = st.Interval
	}

	if st.Timeout <= 0 {
		b.timeout = defaultTimeout
	} else {
		b.timeout = st.Timeout
	}

	if st.ReadyToTrip == nil {
		b.readyToTrip = defaultReadyToTrip
	} else {
		b.readyToTrip = st.ReadyToTrip
	}

	if st.IsSuccessful == nil {
		b.isSuccessful = defaultIsSuccessful
	} else {
		b.isSuccessful = st.IsSuccessful
	}

	b.toNewGeneration(time.Now())
	return b
}

func defaultReadyToTrip(counts Counts) bool {
	return counts.ConsecutiveFailures > 5
}

func defaultIsSuccessful(err error) bool {
	return err == nil
}

func (b *Breaker) toNewGeneration(now time.Time) {
	b.generation++
	b.counts.clear()

	var zero time.Time

	switch b.state {
	case Closed:
		if b.interval == 0 {
			b.expiry = zero
		} else {
			b.expiry = now.Add(b.interval)
		}
	case Open:
		b.expiry = now.Add(b.timeout)
	default:
		b.expiry = zero
	}
}

func (b *Breaker) Name() string {
	return b.name
}

func (b *Breaker) State() State {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	state, _ := b.currentState(now)
	return state
}

func (b *Breaker) currentState(now time.Time) (State, uint64) {
	switch b.state {
	case Closed:
		if !b.expiry.IsZero() && b.expiry.Before(now) {
			b.toNewGeneration(now)
		}
	case Open:
		if b.expiry.Before(now) {
			b.setState(HalfOpen, now)
		}
	}

	return b.state, b.generation
}

func (b *Breaker) setState(state State, now time.Time) {
	if b.state == state {
		return
	}

	prev := b.state
	b.state = state
	b.toNewGeneration(now)

	if b.onStateChange != nil {
		b.onStateChange(b.name, prev, state)
	}
}

func (b *Breaker) Counts() Counts {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.counts
}

func (b *Breaker) Execute(req func() (interface{}, error)) (interface{}, error) {
	generation, err := b.beforeRequest()
	if err != nil {
		return nil, err
	}

	defer func() {
		e := recover()
		if e != nil {
			b.afterRequest(generation, false)
			panic(e)
		}
	}()

	result, err := req()
	b.afterRequest(generation, b.isSuccessful(err))
	return result, err
}

func (b *Breaker) beforeRequest() (uint64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	state, generation := b.currentState(now)

	if state == Open {
		return generation, ErrOpenState
	} else if state == HalfOpen && b.counts.Requests >= b.maxRequests {
		return generation, ErrTooManyRequests
	}

	b.counts.onRequest()
	return generation, nil
}

func (b *Breaker) afterRequest(before uint64, success bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	state, generation := b.currentState(now)
	if generation != before {
		return
	}

	if success {
		b.onSuccess(state, now)
	} else {
		b.onFailure(state, now)
	}
}

func (b *Breaker) onSuccess(state State, now time.Time) {
	switch state {
	case Closed:
		b.counts.onSuccess()
	case HalfOpen:
		b.counts.onSuccess()
		if b.counts.ConsecutiveSuccesses >= b.maxRequests {
			b.setState(Closed, now)
		}
	}
}

func (b *Breaker) onFailure(state State, now time.Time) {
	switch state {
	case Closed:
		b.counts.onFailure()
		if b.readyToTrip(b.counts) {
			b.setState(Open, now)
		}
	case HalfOpen:
		b.setState(Open, now)
	}
}

// TwoStepBreaker 提供一个两步的电路断路器接口
type TwoStepBreaker struct {
	b *Breaker
}

// NewTwoStepBreaker 创建一个新的两步电路断路器
func NewTwoStepBreaker(st Settings) *TwoStepBreaker {
	return &TwoStepBreaker{
		b: NewBreaker(st),
	}
}

func (tb *TwoStepBreaker) Name() string {
	return tb.b.Name()
}

func (tb *TwoStepBreaker) State() State {
	return tb.b.State()
}

func (tb *TwoStepBreaker) Counts() Counts {
	return tb.b.Counts()
}

func (tb *TwoStepBreaker) Allow() (done func(success bool), err error) {
	generation, err := tb.b.beforeRequest()
	if err != nil {
		return nil, err
	}

	return func(success bool) {
		tb.b.afterRequest(generation, success)
	}, nil
}
