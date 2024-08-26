package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"dagger/viztest/internal/dagger"
)

type Viztest struct {
	Num int
}

// LogThroughput logs the current time in a tight loop.
func (m *Viztest) Spam() *dagger.Container {
	for {
		fmt.Println(time.Now())
	}
}

// Encapsulate calls a failing function, but ultimately succeeds.
func (m *Viztest) Encapsulate(ctx context.Context) error {
	_ = m.Fail(ctx, "1")
	return nil // no error, that's the point
}

// FailEffect returns a function whose effects will fail when it runs.
func (m *Viztest) FailEffect() *dagger.Container {
	return dag.Container().
		From("alpine").
		WithExec([]string{"sh", "-exc", "echo 'this is a failing effect' && exit 1"})
}

func (*Viztest) LogStdout() {
	fmt.Println("Hello, world!")
}

func (*Viztest) PrimaryLines(n int) string {
	buf := new(strings.Builder)
	for i := 1; i <= n; i++ {
		fmt.Fprintln(buf, "This is line", i, "of", n)
	}
	return buf.String()
}

func (*Viztest) ManyLines(n int) {
	for i := 1; i <= n; i++ {
		fmt.Println("This is line", i, "of", n)
	}
}

func (vt *Viztest) ManySpans(
	ctx context.Context,
	n int,
	// +default=0
	delayMs int,
) {
	for i := 1; i <= n; i++ {
		_, span := Tracer().Start(ctx, fmt.Sprintf("span %d", i))
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
		span.End()
	}
}

// Continuously prints batches of logs on an interval (default 1 per second).
func (*Viztest) StreamingLogs(
	ctx context.Context,
	// +optional
	// +default=1
	batchSize int,
	// +optional
	// +default=1000
	delayMs int,
) {
	ticker := time.NewTicker(time.Duration(delayMs) * time.Millisecond)
	lineNo := 1
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for i := 0; i < batchSize; i++ {
				fmt.Printf("%d: %d\n", lineNo, time.Now().UnixNano())
				lineNo += 1
			}
		}
	}
}

// Continuously prints batches of logs on an interval (default 1 per second).
func (*Viztest) StreamingChunks(
	ctx context.Context,
	// +optional
	// +default=1
	batchSize int,
	// +optional
	// +default=1000
	delayMs int,
) {
	ticker := time.NewTicker(time.Duration(delayMs) * time.Millisecond)
	lineNo := 1
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for i := 0; i < batchSize; i++ {
				fmt.Printf("%d: %d; ", lineNo, time.Now().UnixNano())
				lineNo += 1
			}
		}
	}
}

func (*Viztest) Echo(ctx context.Context, message string) (string, error) {
	return dag.Container().
		From("alpine").
		WithExec([]string{"echo", message}).
		Stdout(ctx)
}

func (*Viztest) SameDiffClients(ctx context.Context, message string) (string, error) {
	return dag.Container().
		From("alpine").
		WithExec([]string{"sh", "-exc", "for i in $(seq 10); do echo $RANDOM: $0; sleep 1; done", message}).
		Stdout(ctx)
}

// Accounting returns a container that sleeps for 1 second and then sleeps for
// 2 seconds.
//
// It can be used to test UI cues for tracking down the place where a slow
// operation is configured, which is more interesting than the place where it
// is un-lazied when you're trying to figure out where to optimize.
func (*Viztest) Accounting(ctx context.Context) *dagger.Container {
	return dag.Container().
		From("alpine").
		WithEnvVariable("NOW", time.Now().String()).
		WithExec([]string{"sleep", "1"}).
		WithExec([]string{"sleep", "2"})
}

// DeepSleep sleeps forever.
func (*Viztest) DeepSleep(ctx context.Context) *dagger.Container {
	return dag.Container().
		From("alpine").
		WithExec([]string{"sleep", "infinity"})
}

func (v Viztest) Add(
	// +optional
	// +default=1
	diff int,
) *Viztest {
	v.Num++
	return &v
}

func (v Viztest) CountFiles(ctx context.Context, dir *dagger.Directory) (*Viztest, error) {
	ents, err := dir.Entries(ctx)
	if err != nil {
		return nil, err
	}
	v.Num += len(ents)
	return &v, nil
}

func (*Viztest) LogStderr() {
	fmt.Fprintln(os.Stderr, "Hello, world!")
}

// Fail fails after waiting for a certain amount of time.
func (*Viztest) FailLog(ctx context.Context) error {
	_, err := dag.Container().
		From("alpine").
		WithEnvVariable("NOW", time.Now().String()).
		WithExec([]string{"sh", "-c", "echo im doing a lot of work; echo and then failing; exit 1"}).
		Sync(ctx)
	return err
}

// Fail fails after waiting for a certain amount of time.
func (*Viztest) Fail(ctx context.Context,
	// +optional
	// +default="10"
	after string) error {
	_, err := dag.Container().
		From("alpine").
		WithEnvVariable("NOW", time.Now().String()).
		WithExec([]string{"sleep", after}).
		WithExec([]string{"false"}).
		Sync(ctx)
	return err
}

func (*Viztest) Service(ctx context.Context) *dagger.Service {
	return dag.Container().
		From("python").
		WithExposedPort(8000).
		WithExec([]string{"python", "-m", "http.server"}).
		AsService()
}

func (*Viztest) Pending(ctx context.Context) error {
	_, err := dag.Container().
		From("alpine").
		WithExec([]string{"echo", "im cached for good"}).
		WithExec([]string{"echo", "im also cached for good"}).
		WithExec([]string{"echo", "im cached every minute:", time.Now().Truncate(time.Minute).String()}).
		WithExec([]string{"echo", "im busted by that buster"}).
		WithEnvVariable("NOW", time.Now().String()).
		WithExec([]string{"sleep", "1"}).
		WithExec([]string{"sleep", "1"}).
		WithExec([]string{"sleep", "1"}).
		WithExec([]string{"sleep", "1"}).
		WithExec([]string{"sleep", "1"}).
		Sync(ctx)
	return err
}
