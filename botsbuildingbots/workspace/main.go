package main

import (
	"context"
	"dagger/workspace/internal/dagger"
	"dagger/workspace/internal/telemetry"
	_ "embed"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type Workspace struct {
	// +private
	Model string

	// +private
	Attempts int

	// The current system prompt.
	SystemPrompt string
}

var knownModels = []string{
	"gpt-4o",
	"gemini-2.0-flash",
	"claude-3-5-sonnet-latest",
	"claude-3-7-sonnet-latest",
}

type EvalFunc = func(*dagger.Evals) *dagger.EvalsReport

var evals = map[string]EvalFunc{
	"BuildMulti":            (*dagger.Evals).BuildMulti,
	"BuildMultiNoVar":       (*dagger.Evals).BuildMultiNoVar,
	"ReadImplicitVars":      (*dagger.Evals).ReadImplicitVars,
	"SingleState":           (*dagger.Evals).SingleState,
	"SingleStateTransition": (*dagger.Evals).SingleStateTransition,
	"UndoSingle":            (*dagger.Evals).UndoSingle,
}

func New(
	// +default=""
	model string,
	// +default=2
	attempts int,
	// +default=""
	systemPrompt string,
) *Workspace {
	return &Workspace{
		Model:        model,
		Attempts:     attempts,
		SystemPrompt: systemPrompt,
	}
}

// Set the system prompt for future evaluations.
func (w *Workspace) WithSystemPrompt(prompt string) *Workspace {
	w.SystemPrompt = prompt
	return w
}

// Backoff sleeps for the given duration in seconds.
//
// Use this if you're getting rate limited.
func (w *Workspace) Backoff(seconds int) *Workspace {
	time.Sleep(time.Duration(seconds) * time.Second)
	return w
}

// The list of possible evals you can run.
func (w *Workspace) EvalNames() []string {
	var names []string
	for eval := range evals {
		names = append(names, eval)
	}
	sort.Strings(names)
	return names
}

// Run an evaluation and return its report.
func (w *Workspace) Evaluate(ctx context.Context, eval string) (string, error) {
	evalFn, ok := evals[eval]
	if !ok {
		return "", fmt.Errorf("unknown evaluation: %s", eval)
	}
	reports := make([]string, w.Attempts)
	wg := new(sync.WaitGroup)
	var successCount int
	for attempt := range w.Attempts {
		wg.Add(1)
		go func() {
			defer wg.Done()

			ctx, span := Tracer().Start(ctx, fmt.Sprintf("attempt %d", attempt+1),
				telemetry.Reveal())

			var rerr error
			defer telemetry.End(span, func() error { return rerr })

			report := new(strings.Builder)
			defer func() { reports[attempt] = report.String() }()

			fmt.Fprintf(report, "## Attempt %d\n", attempt+1)
			fmt.Fprintln(report)

			eval := w.evaluate(attempt, evalFn)

			evalReport, err := eval.Report(ctx)
			if err != nil {
				rerr = err
				return
			}
			fmt.Fprintln(report, evalReport)

			succeeded, err := eval.Succeeded(ctx)
			if err != nil {
				rerr = err
				return
			}
			if succeeded {
				successCount++
			}
		}()
	}

	wg.Wait()

	finalReport := new(strings.Builder)
	fmt.Fprintln(finalReport, "# Model:", w.Model)
	fmt.Fprintln(finalReport)
	fmt.Fprintln(finalReport, "## All Attempts")
	fmt.Fprintln(finalReport)
	for _, report := range reports {
		fmt.Fprint(finalReport, report)
	}

	fmt.Fprintln(finalReport, "## Final Report")
	fmt.Fprintln(finalReport)
	fmt.Fprintf(finalReport, "SUCCESS RATE: %d/%d (%.f%%)\n", successCount, w.Attempts, float64(successCount)/float64(w.Attempts)*100)

	return finalReport.String(), nil
}

// Run an evaluation across all known models in parallel.
func (w *Workspace) EvaluateAllModelsOnce(ctx context.Context, name string) ([]string, error) {
	reports := make([]string, len(knownModels))
	wg := new(sync.WaitGroup)
	for i, model := range knownModels {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, span := Tracer().Start(ctx, fmt.Sprintf("model: %s", model),
				telemetry.Reveal())
			report, err := New(model, 1, w.SystemPrompt).Evaluate(ctx, name)
			telemetry.End(span, func() error { return err })
			if err != nil {
				reports[i] = fmt.Sprintf("ERROR: %s", err)
			} else {
				reports[i] = report
			}
		}()
	}
	wg.Wait()
	return reports, nil
}

func (w *Workspace) evaluate(attempt int, evalFn EvalFunc) *dagger.EvalsReport {
	return evalFn(
		dag.Evals().
			WithAttempt(attempt + 1).
			WithModel(w.Model).
			WithSystemPrompt(w.SystemPrompt),
	)
}
