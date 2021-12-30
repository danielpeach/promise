package promise

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAwaitSuccess(t *testing.T) {
	promise := New[int](context.Background(), func(ctx context.Context) (*int, error) {
		time.Sleep(1 * time.Millisecond)
		ret := 100
		return &ret, nil
	})

	value, err := promise.Await()
	assert.NoError(t, err)
	assert.Equal(t, 100, *value)
}

func TestAwaitTwice(t *testing.T) {
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
	case <-timer.C:
		assert.Fail(t, "test timed out")
	case <-doneChan:
		assert.Equal(t, 100, *value)
	case <-errChan:
		assert.NoError(t, err)
	}
}

func TestParallelAwait(t *testing.T) {
	promise := New[int](context.Background(), func(ctx context.Context) (*int, error) {
		ret := 100
		return &ret, nil
	})

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		value, err := promise.Await()
		assert.NoError(t, err)
		assert.Equal(t, 100, *value)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		value, err := promise.Await()
		assert.NoError(t, err)
		assert.Equal(t, 100, *value)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		value, err := promise.Await()
		assert.NoError(t, err)
		assert.Equal(t, 100, *value)
		wg.Done()
	}()

	wg.Wait()
}

func TestCancelPromise(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	promise := New[int](ctx, func(ctx context.Context) (*int, error) {
		time.Sleep(1 * time.Second)
		ret := 1
		return &ret, nil
	})

	cancel()
	value, err := promise.Await()
	assert.Nil(t, value)
	assert.Equal(t, "context canceled", err.Error())
}

func TestAwaitError(t *testing.T) {
	promise := New[int](context.Background(), func(ctx context.Context) (*int, error) {
		time.Sleep(1 * time.Millisecond)
		return nil, fmt.Errorf("whoops")
	})
	value, err := promise.Await()
	assert.Nil(t, value)
	assert.Equal(t, "whoops", err.Error())
}

func TestThen(t *testing.T) {
	first := New[int](context.Background(), func(ctx context.Context) (*int, error) {
		val := 1
		return &val, nil
	})

	second := Then[int, string](first, func(ctx context.Context, value *int, err error) (*string, error) {
		str := strconv.Itoa(*value)
		return &str, nil
	})

	secondValue, err := second.Await()
	assert.NoError(t, err)
	assert.Equal(t, "1", *secondValue)

	firstValue, err := first.Await()
	assert.NoError(t, err)
	assert.Equal(t, 1, *firstValue)
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

	values, err := All(a, b, c).Await()
	assert.Nil(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, *values)
}

func TestAllThen(t *testing.T) {
	a := New[int](context.Background(), func(ctx context.Context) (*int, error) {
		val := 1
		return &val, nil
	})

	b := New[int](context.Background(), func(ctx context.Context) (*int, error) {
		val := 2
		return &val, nil
	})

	value, err := Then[[]int, int](All(a, b), func(ctx context.Context, value *[]int, err error) (*int, error) {
		slice := *value
		added := slice[0] + slice[1]
		return &added, nil
	}).Await()

	assert.Nil(t, err)
	assert.Equal(t, 3, *value)
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

	values, err := All(a, b, c).Await()
	assert.Nil(t, values)
	assert.Equal(t, "whoops", err.Error())
}
