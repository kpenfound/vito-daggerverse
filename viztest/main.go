package main

import (
	"context"
	"fmt"
	"os"
	"time"
)

type Viztest struct {
	Num int
}

// LogThroughput logs the current time in a tight loop.
func (m *Viztest) Spam() *Container {
	for {
		fmt.Println(time.Now())
	}
}

// Encapsulate calls a failing function, but ultimately succeeds.
func (m *Viztest) Encapsulate(ctx context.Context) error {
	_ = m.Fail(ctx, "1")
	return nil
}

func (*Viztest) LogStdout() {
	fmt.Println("Hello, world!")
}

func (*Viztest) Echo(ctx context.Context, message string) (string, error) {
	return dag.Container().
		From("alpine").
		WithExec([]string{"echo", message}).
		Stdout(ctx)
}

// Accounting returns a container that sleeps for 1 second and then sleeps for
// 2 seconds.
//
// It can be used to test UI cues for tracking down the place where a slow
// operation is configured, which is more interesting than the place where it
// is un-lazied when you're trying to figure out where to optimize.
func (*Viztest) Accounting(ctx context.Context) *Container {
	return dag.Container().
		From("alpine").
		WithEnvVariable("NOW", time.Now().String()).
		WithExec([]string{"sleep", "1"}).
		WithExec([]string{"sleep", "2"})
}

func (v Viztest) Add(
	// +optional
	// +default=1
	diff int,
) *Viztest {
	v.Num++
	return &v
}

func (v Viztest) CountFiles(ctx context.Context, dir *Directory) (*Viztest, error) {
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

func (*Viztest) CustomSpan(ctx context.Context) {
	ctx, span := Tracer().Start(ctx, "span1")
	defer span.End()
}
