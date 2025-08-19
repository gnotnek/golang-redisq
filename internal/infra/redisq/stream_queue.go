package redisq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"redisq/internal/domain"
	"redisq/internal/ports"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var _ ports.Queue = (*Client)(nil)

func (c *Client) Enqueue(ctx context.Context, t domain.Task) (string, error) {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}

	t.Status = domain.StatusQueued
	t.CreatedAt = time.Now()
	b, _ := json.Marshal(t)
	id, err := c.Rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: c.Cfg.StreamKey,
		Values: map[string]interface{}{"task": b},
	}).Result()

	if err != nil {
		return "", err
	}
	_ = c.SaveState(ctx, t)
	return id, nil
}

func (c *Client) EnqueueDelayed(ctx context.Context, t domain.Task, runAt time.Time) (string, error) {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	t.Status = domain.StatusDelayed
	t.NextRunAt = runAt
	if err := c.SaveState(ctx, t); err != nil {
		return "", err
	}
	score := float64(runAt.UnixMilli())
	if err := c.Rdb.ZAdd(ctx, c.Cfg.ScheduledZSet, redis.Z{Score: score,
		Member: t.ID}).Err(); err != nil {
		return "", err
	}
	return t.ID, nil
}

func (c *Client) Claim(ctx context.Context, consumer string, block time.Duration) (*domain.Task, string, error) {
	res, err := c.Rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    c.Cfg.Group,
		Consumer: consumer,
		Streams:  []string{c.Cfg.StreamKey, ">"},
		Count:    1,
		Block:    block,
	}).Result()

	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, "", nil
		}
		return nil, "", err
	}

	if len(res) == 0 || len(res[0].Messages) == 0 {
		return nil, "", nil
	}

	msg := res[0].Messages[0]
	raw := msg.Values["task"]
	var t domain.Task
	switch v := raw.(type) {
	case string:
		_ = json.Unmarshal([]byte(v), &t)
	case []byte:
		_ = json.Unmarshal(v, &t)
	default:
		return nil, "", fmt.Errorf("unexpected task type: %T", v)
	}
	return &t, msg.ID, nil
}

func (c *Client) Ack(ctx context.Context, streamID string) error {
	return c.Rdb.XAck(ctx, c.Cfg.StreamKey, c.Cfg.Group, streamID).Err()
}

func (c *Client) Fail(ctx context.Context, streamID string, t domain.Task, err error) error {
	t.Attempts++
	_ = c.SaveState(ctx, t)
	return nil
}

func (c *Client) ToDLQ(ctx context.Context, streamID string, t domain.Task, reason string) error {
	b, _ := json.Marshal(struct {
		domain.Task
		Reason string `json:"reason"`
	}{t, reason})
	if err := c.Rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: c.Cfg.DLQStreamKey,
		Values: map[string]interface{}{"task": b},
	}).Err(); err != nil {
		return err
	}

	_ = c.Rdb.XAck(ctx, c.Cfg.StreamKey, c.Cfg.Group, streamID).Err()
	t.Status = domain.StatusFailed
	return c.SaveState(ctx, t)
}

func (c *Client) SaveState(ctx context.Context, t domain.Task) error {
	key := fmt.Sprintf("task:%s", t.ID)
	m := map[string]any{
		"status":       string(t.Status),
		"attempts":     t.Attempts,
		"max_attempts": t.MaxAttempts,
		"type":         t.Type,
		"next_run_at":  t.NextRunAt.UnixMilli(),
	}
	for k, v := range t.Payload {
		m["payload:"+k] = v
	}
	return c.Rdb.HSet(ctx, key, m).Err()
}

func (c *Client) Get(ctx context.Context, id string) (*domain.Task, error) {
	key := fmt.Sprintf("task:%s", id)
	h, err := c.Rdb.HGetAll(ctx, key).Result()
	if err != nil || len(h) == 0 {
		return nil, err
	}

	t := &domain.Task{
		ID:      id,
		Payload: map[string]string{},
	}
	t.Type = h["type"]
	t.Status = domain.TaskStatus(h["status"])

	for k, v := range h {
		if len(k) > 8 && k[:8] == "payload:" {
			t.Payload[k[8:]] = v
		}
	}
	return t, nil
}
