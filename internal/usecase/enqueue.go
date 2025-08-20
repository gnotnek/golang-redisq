package usecase

import (
	"context"
	"redisq/internal/domain"
	"redisq/internal/ports"
	"time"
)

type Enqueuer struct {
	Q ports.Queue
}

func (e Enqueuer) Now(ctx context.Context, t domain.Task) (string, error) {
	if t.MaxAttempts == 0 {
		t.MaxAttempts = 5
	}
	return e.Q.Enqueue(ctx, t)
}

func (e Enqueuer) At(ctx context.Context, t domain.Task, runAt time.Time) (string, error) {
	if t.MaxAttempts == 0 {
		t.MaxAttempts = 5
	}
	return e.Q.EnqueueDelayed(ctx, t, runAt)
}
