package breaker

import (
	"errors"
	"sync"
	"time"
)

type State int

const (
	StateClosed State = iota
	StateHalfOpen
	StateOpen
)

const defaultInterval = time.Duration(0) * time.Second
const defaultTimeout = time.Duration(60) * time.Second

var (
	ErrTooManyRequests = errors.New("too many requests")
	ErrOpenState       = errors.New("circuit breaker is open")
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateHalfOpen:
		return "half-open"
	case StateOpen:
		return "open"
	default:
		return "unknown state"
	}
}

type Counts struct {
	Requests             uint32
	TotalSuccesses       uint32
	TotalFailures        uint32
	ConsecutiveSuccesses uint32 // 连续成功次数
	ConsecutiveFailures  uint32 // 连续失败次数
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

type Settings struct {
	Name          string
	MaxRequests   uint32
	Interval      time.Duration
	Timeout       time.Duration
	ReadyToTrip   func(counts Counts) bool
	OnStateChange func(name string, from State, to State)
	IsSuccessful  func(err error) bool
}

type Breaker[T any] struct {
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

func NewBreaker[T any](st Settings) *Breaker[T] {
	b := new(Breaker[T])

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

func (b *Breaker[T]) toNewGeneration(now time.Time) {
	b.generation++
	b.counts.clear()

	var zero time.Time

	switch b.state {
	case StateClosed:
		if b.interval == 0 {
			b.expiry = zero
		} else {
			b.expiry = now.Add(b.interval)
		}
	case StateOpen:
		b.expiry = now.Add(b.timeout)
	default:
		b.expiry = zero
	}
}

func (b *Breaker[T]) Name() string {
	return b.name
}

func (b *Breaker[T]) State() State {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	state, _ := b.currentState(now)
	return state
}

func (b *Breaker[T]) currentState(now time.Time) (State, uint64) {
	switch b.state {
	case StateClosed:
		if !b.expiry.IsZero() && b.expiry.Before(now) {
			b.toNewGeneration(now)
		}
	case StateOpen:
		if b.expiry.Before(now) {
			b.setState(StateHalfOpen, now)
		}
	}

	return b.state, b.generation
}

func (b *Breaker[T]) setState(state State, now time.Time) {
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

func (b *Breaker[T]) Counts() Counts {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.counts
}

func (b *Breaker[T]) Execute(req func() (T, error)) (T, error) {
	generation, err := b.beforeRequest()
	if err != nil {
		var defaultValue T
		return defaultValue, err
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

func (b *Breaker[T]) beforeRequest() (uint64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	state, generation := b.currentState(now)

	if state == StateOpen {
		return generation, ErrOpenState
	} else if state == StateHalfOpen && b.counts.Requests >= b.maxRequests {
		return generation, ErrTooManyRequests
	}

	b.counts.onRequest()
	return generation, nil
}

func (b *Breaker[T]) afterRequest(before uint64, success bool) {
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

func (b *Breaker[T]) onSuccess(state State, now time.Time) {
	switch state {
	case StateClosed:
		b.counts.onSuccess()
	case StateHalfOpen:
		b.counts.onSuccess()
		if b.counts.ConsecutiveSuccesses >= b.maxRequests {
			b.setState(StateClosed, now)
		}
	}
}

func (b *Breaker[T]) onFailure(state State, now time.Time) {
	switch state {
	case StateClosed:
		b.counts.onFailure()
		if b.readyToTrip(b.counts) {
			b.setState(StateOpen, now)
		}
	case StateHalfOpen:
		b.setState(StateOpen, now)
	}
}

type TwoStepBreaker[T any] struct {
	b *Breaker[T]
}

func NewTwoStepBreaker[T any](st Settings) *TwoStepBreaker[T] {
	return &TwoStepBreaker[T]{
		b: NewBreaker[T](st),
	}
}

func (tb *TwoStepBreaker[T]) Name() string {
	return tb.b.Name()
}

func (tb *TwoStepBreaker[T]) State() State {
	return tb.b.State()
}

func (tb *TwoStepBreaker[T]) Counts() Counts {
	return tb.b.Counts()
}

func (tb *TwoStepBreaker[T]) Allow() (done func(success bool), err error) {
	generation, err := tb.b.beforeRequest()
	if err != nil {
		return nil, err
	}

	return func(success bool) {
		tb.b.afterRequest(generation, success)
	}, nil
}
