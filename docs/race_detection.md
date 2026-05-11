-# Race Detection

The integration tests run with `-race`, which instruments the binary to detect data races at runtime. It only catches races that actually execute, so coverage determines effectiveness.

---

## Current Setup

`just itest` runs `go test -race ./queue/...` against a real Postgres instance. CGO must be enabled (the default) for the race detector to work.

---

## Improvements

### 1. Concurrent integration test for `Add()`

The most important race to verify is two simultaneous signups computing the same insertion position. A test that fires two goroutines calling `Add()` at the same time and then asserts the resulting queue has correct, distinct positions would directly exercise the `SELECT FOR UPDATE` locking added in the signup ordering implementation.

```go
func TestAdd_ConcurrentSignups(t *testing.T) {
    // fire two Add() calls simultaneously via goroutines
    // assert both entries are in the queue
    // assert their positions are distinct
}
```

### 2. Run tests multiple times

Timing-dependent races don't always manifest on a single run. `-count=N` reruns the tests N times, increasing the chance a race surfaces:

```bash
go test -race -count=5 ./queue/...
```

Not needed in CI on every push, but useful when actively working on concurrent code.

### 3. Load testing

`dev_tools/add_users_to_queue.hurl` already hits `/signup`. Running it with concurrent workers would surface races in the live app that integration tests don't cover. Consider adding a concurrent variant to `dev_tools/` for manual use before releases.