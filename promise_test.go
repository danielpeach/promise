package promise

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
	"time"
)

func TestPromiseSuccess(t *testing.T) {
	promise := New[int](context.Background(), func(ctx context.Context) (*int, error) {
		time.Sleep(1 * time.Millisecond)
		ret := 100
		return &ret, nil
	})

	value, err := promise.Await()
	assert.NoError(t, err)
	assert.Equal(t, 100, *value)
}

func TestPromiseAwaitTwice(t *testing.T) {
	promise := New[int](context.Background(), func(ctx context.Context) (*int, error) {
		time.Sleep(30 * time.Millisecond)
		ret := 100
		return &ret, nil
	})

	value, err := promise.Await()
	assert.NoError(t, err)
	assert.Equal(t, 100, *value)

	doneChan := make(chan *int)
	errChan := make(chan error)
	timer := time.NewTimer(10 * time.Millisecond)
	defer timer.Stop()

	go func() {
		value, err := promise.Await()
		if err != nil {
			errChan <- err
			return
		}
		doneChan <- value
	}()

	select {
	case <- timer.C:
		assert.Fail(t, "test timed out")
	case <- doneChan:
		assert.Equal(t, 100, *value)
	case <- errChan:
		assert.NoError(t, err)
	}
}

func TestPromiseError(t *testing.T) {
	promise := New[int](context.Background(), func(ctx context.Context) (*int, error) {
		time.Sleep(1 * time.Millisecond)
		return nil, fmt.Errorf("whoops")
	})
	value, err := promise.Await()
	assert.Nil(t, value)
	assert.Equal(t, "whoops", err.Error())
}

func TestThen(t *testing.T) {
	promise := New[int](context.Background(), func(ctx context.Context) (*int, error) {
		val := 1
		return &val, nil
	})

	value, err := Then[int, string](promise, func(ctx context.Context, value *int, err error) (*string, error) {
		str := strconv.Itoa(*value)
		return &str, nil
	}).Await()

	assert.NoError(t, err)
	assert.Equal(t, "1", *value)
}

func TestThenError(t *testing.T) {
	promise := New[int](context.Background(), func(ctx context.Context) (*int, error) {
		time.Sleep(1 * time.Millisecond)
		return nil, fmt.Errorf("whoops")
	})

	value, err := Then[int, string](promise, func(ctx context.Context, value *int, err error) (*string, error) {
		return nil, err
	}).Await()

	assert.Nil(t, value)
	assert.Equal(t, "whoops", err.Error())
}

func TestAll(t *testing.T) {
	a := New[string](context.Background(), func(ctx context.Context) (*string, error) {
		time.Sleep(100 * time.Millisecond)
		ret := "a"
		return &ret, nil
	})

	b := New[string](context.Background(), func(ctx context.Context) (*string, error) {
		time.Sleep(1 * time.Millisecond)
		ret := "b"
		return &ret, nil
	})

	c := New[string](context.Background(), func(ctx context.Context) (*string, error) {
		time.Sleep(10 * time.Millisecond)
		ret := "c"
		return &ret, nil
	})

	values, err := All(a, b, c)
	assert.Nil(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, values)
}

func TestAllError(t *testing.T) {
	a := New[string](context.Background(), func(ctx context.Context) (*string, error) {
		time.Sleep(100 * time.Millisecond)
		return nil, fmt.Errorf("whoops")
	})

	b := New[string](context.Background(), func(ctx context.Context) (*string, error) {
		time.Sleep(1 * time.Millisecond)
		ret := "b"
		return &ret, nil
	})

	c := New[string](context.Background(), func(ctx context.Context) (*string, error) {
		time.Sleep(1 * time.Millisecond)
		ret := "c"
		return &ret, nil
	})

	values, err := All(a, b, c)
	assert.Nil(t, values)
	assert.Equal(t, "whoops", err.Error())
}
