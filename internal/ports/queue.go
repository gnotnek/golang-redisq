package ports

import (
	"context"
	"redisq/internal/domain"
	"time"
)

type Queue interface {
	Enqueue(ctx context.Context, t domain.Task) (string, error)
	EnqueueDelayed(ctx context.Context, t domain.Task, runAt time.Time) (string, error)
	Claim(ctx context.Context, consumer string, block time.Duration) (*domain.Task, string /*streamID*/, error)
	Ack(ctx context.Context, streamID string) error
	Fail(ctx context.Context, streamID string, t domain.Task, err error) error
	ToDLQ(ctx context.Context, streamID string, t domain.Task, reason string) error
	SaveState(ctx context.Context, t domain.Task) error
	Get(ctx context.Context, id string) (*domain.Task, error)
}

type Scheduler interface {
	// moves due tasks from ZSET into the stream
	Run(ctx context.Context) error
}
