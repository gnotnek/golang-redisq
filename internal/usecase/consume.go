package usecase

import (
	"context"
	"redisq/internal/domain"
	"redisq/internal/ports"
	"redisq/pkg/backoff"
	"time"
)

type Handler func(ctx context.Context, t domain.Task) error

type Consumer struct {
	Q            ports.Queue
	ConsumerName string
	BaseBackoff  time.Duration
	MaxBackoff   time.Duration
}

func (c Consumer) Run(ctx context.Context, handle Handler) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		t, id, err := c.Q.Claim(ctx, c.ConsumerName, 5*time.Second)
		if err != nil {
			continue
		}
		if t == nil {
			continue
		}

		// Mark running
		t.Status = domain.StatusRunning
		_ = c.Q.SaveState(ctx, *t)

		err = handle(ctx, *t)
		if err == nil {
			_ = c.Q.Ack(ctx, id)
			t.Status = domain.StatusDone
			_ = c.Q.SaveState(ctx, *t)
			continue
		}

		// Failure path: retry or DLQ
		if t.Attempts+1 >= t.MaxAttempts {
			_ = c.Q.ToDLQ(ctx, id, *t, err.Error())
			continue
		}

		// compute backoff and reschedule by adding back to scheduled ZSET via SaveState + ZADD
		delay := backoff.ExponentialJitter(c.BaseBackoff, c.MaxBackoff, t.Attempts+1)
		t.NextRunAt = time.Now().Add(delay)
		_ = c.Q.Fail(ctx, id, *t, err)

		// remove from PEL by acking and then re-inserting as delayed
		_ = c.Q.Ack(ctx, id)
		_, _ = c.Q.EnqueueDelayed(ctx, *t, t.NextRunAt)
	}
}
