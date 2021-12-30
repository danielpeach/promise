# Go Promises
Go Promises is inspired by JavaScript's [`Promise` implementation](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Promise).

## Usage

### Parallel Fetch

Make IO-bound calls in parallel. Block on their completion without touching goroutines or channels.
```go
promiseForSnakes := promise.New[[]snake](ctx, func(ctx context.Context) ([]snake, error) {
	// Fetch some snakes!
	// This function will launch immediately but promise.New(...) won't block on me!
})

promiseForBees := promise.New[[]bee](ctx, func(ctx context.Context) ([]bee, error) {
	// Fetch some bees! 
	// This function will launch immediately but promise.New(...) won't block on me!
})

snakes, err := promiseForSnakes.Await()
// Handle snake error (e.g., teeth too sharp!)

bees, err := promiseForBees.Await()
// Handle bee error (e.g., stingers too pointy!)
```

### `promise.All`

Wait until all your promises have completed.

```go

hats, err := promise.All[hat](
	ctx, 
	promise.New[hat](ctx, func(ctx context.Context) (hat, error) {
		// Fetch ten-gallon hat.
	}), 
	promise.New[hat](ctx, func(ctx context.Context) (hat, error) { 
		// Fetch fedora.
	}), 
	promise.New[hat](ctx, func(ctx context.Context) (hat, error) { 
		// Fetch balaclava.
	}),
).Await()

```