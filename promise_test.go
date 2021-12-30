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
	promise := New[int](context.Background(), func(ctx context.Context) (int, error) {
		time.Sleep(1 * time.Millisecond)
		return 100, nil
	})

	value, err := promise.Await()
	assert.NoError(t, err)
	assert.Equal(t, 100, value)
}

func TestAwaitTwice(t *testing.T) {
	promise := New[int](context.Background(), func(ctx context.Context) (int, error) {
		time.Sleep(30 * time.Millisecond)
		return 100, nil
	})

	value, err := promise.Await()
	assert.NoError(t, err)
	assert.Equal(t, 100, value)

	doneChan := make(chan int)
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
		assert.Equal(t, 100, value)
	case <-errChan:
		assert.NoError(t, err)
	}
}

func TestParallelAwait(t *testing.T) {
	promise := New[int](context.Background(), func(ctx context.Context) (int, error) {
		time.Sleep(1 * time.Millisecond)
		return 100, nil
	})

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		value, err := promise.Await()
		assert.NoError(t, err)
		assert.Equal(t, 100, value)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		value, err := promise.Await()
		assert.NoError(t, err)
		assert.Equal(t, 100, value)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		value, err := promise.Await()
		assert.NoError(t, err)
		assert.Equal(t, 100, value)
		wg.Done()
	}()

	wg.Wait()
}

func TestChainedThenWithError(t *testing.T) {
	failed := New[int](context.Background(), func(ctx context.Context) (int, error) {
		return 0, fmt.Errorf("whoops")
	})

	value, err := Then[int, int](failed, func(ctx context.Context, value int, err error) (int, error) {
		assert.Equal(t, "whoops", err.Error())
		return 1, nil
	}).Await()

	assert.NoError(t, err)
	assert.Equal(t, 1, value)
}

func TestCancelPromise(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	promise := New[int](ctx, func(ctx context.Context) (int, error) {
		time.Sleep(1 * time.Second)
		return 1, nil
	})

	cancel()
	_, err := promise.Await()
	assert.Equal(t, "context canceled", err.Error())
}

func TestContextPropagation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	doneChan := make(chan struct{})

	New[int](ctx, func(ctx context.Context) (int, error) {
		<-ctx.Done()
		doneChan <- struct{}{}
		return 0, nil
	})

	cancel()

	timer := time.NewTimer(10 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-timer.C:
		assert.Fail(t, "test timed out")
	case <-doneChan:
	}
}

func TestResolve(t *testing.T) {
	value, err := Resolve[string]("resolved").Await()
	assert.NoError(t, err)
	assert.Equal(t, "resolved", value)
}

func TestResolveThen(t *testing.T) {
	value, err := Then[int, int](Resolve[int](1), func(ctx context.Context, value int, err error) (int, error) {
		return value + 2, nil
	}).Await()

	assert.NoError(t, err)
	assert.Equal(t, 3, value)
}

func TestReject(t *testing.T) {
	_, err := Reject[string](fmt.Errorf("whoops")).Await()
	
	assert.Equal(t, "whoops", err.Error())
}

func TestAwaitError(t *testing.T) {
	promise := New[int](context.Background(), func(ctx context.Context) (int, error) {
		time.Sleep(1 * time.Millisecond)
		return 0, fmt.Errorf("whoops")
	})
	_, err := promise.Await()
	assert.Equal(t, "whoops", err.Error())
}

func TestThen(t *testing.T) {
	first := New[int](context.Background(), func(ctx context.Context) (int, error) {
		return 1, nil
	})

	second := Then[int, string](first, func(ctx context.Context, value int, err error) (string, error) {
		return strconv.Itoa(value), nil
	})

	secondValue, err := second.Await()
	assert.NoError(t, err)
	assert.Equal(t, "1", secondValue)

	firstValue, err := first.Await()
	assert.NoError(t, err)
	assert.Equal(t, 1, firstValue)
}

func TestThenError(t *testing.T) {
	promise := New[int](context.Background(), func(ctx context.Context) (int, error) {
		time.Sleep(1 * time.Millisecond)
		return 0, fmt.Errorf("whoops")
	})

	_, err := Then[int, string](promise, func(ctx context.Context, value int, err error) (string, error) {
		return "", err
	}).Await()

	assert.Equal(t, "whoops", err.Error())
}

func TestAll(t *testing.T) {
	a := New[string](context.Background(), func(ctx context.Context) (string, error) {
		time.Sleep(100 * time.Millisecond)
		return "a", nil
	})

	b := New[string](context.Background(), func(ctx context.Context) (string, error) {
		time.Sleep(1 * time.Millisecond)
		return "b", nil
	})

	c := New[string](context.Background(), func(ctx context.Context) (string, error) {
		time.Sleep(10 * time.Millisecond)
		return "c", nil
	})

	values, err := All(context.Background(), a, b, c).Await()
	assert.Nil(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, values)
}

func TestAllThen(t *testing.T) {
	a := New[int](context.Background(), func(ctx context.Context) (int, error) {
		return 1, nil
	})

	b := New[int](context.Background(), func(ctx context.Context) (int, error) {
		return 2, nil
	})

	value, err := Then[[]int, int](All(context.Background(), a, b), func(ctx context.Context, value []int, err error) (int, error) {
		return value[0] + value[1], nil
	}).Await()

	assert.Nil(t, err)
	assert.Equal(t, 3, value)
}

func TestAllError(t *testing.T) {
	a := New[string](context.Background(), func(ctx context.Context) (string, error) {
		time.Sleep(100 * time.Millisecond)
		return "", fmt.Errorf("whoops")
	})

	b := New[string](context.Background(), func(ctx context.Context) (string, error) {
		time.Sleep(1 * time.Millisecond)
		return "b", nil
	})

	c := New[string](context.Background(), func(ctx context.Context) (string, error) {
		time.Sleep(1 * time.Millisecond)
		return "c", nil
	})

	values, err := All(context.Background(), a, b, c).Await()
	assert.Nil(t, values)
	assert.Equal(t, "whoops", err.Error())
}

func TestRace(t *testing.T) {
	winner := New[string](context.Background(), func(ctx context.Context) (string, error) {
		return "winner", nil
	})

	loser := New[string](context.Background(), func(ctx context.Context) (string, error) {
		time.Sleep(time.Second)
		return "loser", nil
	})

	value, err := Race[string](context.Background(), winner, loser).Await()
	assert.Nil(t, err)
	assert.Equal(t, "winner", value)
}

func TestRaceError(t *testing.T) {
	a := New[string](context.Background(), func(ctx context.Context) (string, error) {
		return "", fmt.Errorf("whoops")
	})

	b := New[string](context.Background(), func(ctx context.Context) (string, error) {
		time.Sleep(time.Millisecond)
		return "b", nil
	})

	_, err := Race[string](context.Background(), a, b).Await()
	
	assert.Equal(t, "whoops", err.Error())
}

func TestRaceCancel(t *testing.T) {
	a := New[string](context.Background(), func(ctx context.Context) (string, error) {
		time.Sleep(10 * time.Millisecond)
		return "a", nil
	})

	b := New[string](context.Background(), func(ctx context.Context) (string, error) {
		time.Sleep(10 * time.Millisecond)
		return "b", nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	racer := Race[string](ctx, a, b)

	cancel()

	_, err := racer.Await()
	assert.Equal(t, "context canceled", err.Error())
}

func TestAny(t *testing.T) {
	a := New[string](context.Background(), func(ctx context.Context) (string, error) {
		time.Sleep(100 * time.Millisecond)
		return "", fmt.Errorf("a failed")
	})

	b := New[string](context.Background(), func(ctx context.Context) (string, error) {
		time.Sleep(10 * time.Millisecond)
		return "b succeeded", nil
	})

	c := New[string](context.Background(), func(ctx context.Context) (string, error) {
		return "", fmt.Errorf("c failed")
	})

	value, err := Any[string](context.Background(), a, b, c).Await()
	assert.Nil(t, err)
	assert.Equal(t, "b succeeded", value)
}

func TestAnyError(t *testing.T) {
	a := New[string](context.Background(), func(ctx context.Context) (string, error) {
		time.Sleep(100 * time.Millisecond)
		return "", fmt.Errorf("a failed")
	})

	b := New[string](context.Background(), func(ctx context.Context) (string, error) {
		return "", fmt.Errorf("b failed")
	})

	c := New[string](context.Background(), func(ctx context.Context) (string, error) {
		return "", fmt.Errorf("c failed")
	})

	_, err := Any[string](context.Background(), a, b, c).Await()
	assert.Contains(t, err.Error(), "a failed, b failed, c failed")
}