package runner

import (
	"context"

	"github.com/yazanabuashour/openstudy/internal/localruntime"
	"github.com/yazanabuashour/openstudy/internal/study"
)

func withRuntime[T any](ctx context.Context, config Config, fn func(*localruntime.Runtime) (T, error)) (T, error) {
	runtime, err := localruntime.Open(ctx, localruntime.Config(config))
	if err != nil {
		var zero T
		return zero, err
	}
	defer func() {
		_ = runtime.Close()
	}()
	return fn(runtime)
}

func withStudyService[T any](ctx context.Context, config Config, fn func(*study.Service) (T, error)) (T, error) {
	return withRuntime(ctx, config, func(runtime *localruntime.Runtime) (T, error) {
		return fn(runtime.Service)
	})
}

func rejectedBase(reason string) BaseResult {
	return BaseResult{
		Rejected:        true,
		RejectionReason: reason,
		Summary:         reason,
	}
}
