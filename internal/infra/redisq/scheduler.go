package redisq

import (
	"context"
	"encoding/json"
	"redisq/internal/domain"
	"redisq/internal/ports"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

var _ ports.Scheduler = (*Scheduler)(nil)

type Scheduler struct {
	C        *Client
	Interval time.Duration
}

func NewScheduler(c *Client, interval time.Duration) *Scheduler {
	return &Scheduler{C: c, Interval: interval}
}

func (s *Scheduler) Run(ctx context.Context) error {
	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := s.moveDue(ctx); err != nil {
				log.Ctx(ctx).Error().Err(err).Msg("scheduler moveDue failed")
				// Don't exit, just log and continue
			}
		}
	}
}

func (s *Scheduler) moveDue(ctx context.Context) error {
	now := nowMs()
	ids, err := s.C.Rdb.ZRangeByScore(ctx, s.C.Cfg.ScheduledZSet, &redis.ZRangeBy{
		Min:    "-inf",
		Max:    fmtFloat(now),
		Offset: 0,
		Count:  128,
	}).Result()
	if err != nil {
		return err // only return actual Redis error
	}
	if len(ids) == 0 {
		return nil // nothing to move this tick
	}

	for _, id := range ids {
		key := "task:" + id
		h, err := s.C.Rdb.HGetAll(ctx, key).Result()
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Str("task_id", id).Msg("failed to fetch task hash")
			continue
		}

		t := domain.Task{ID: id, Type: h["type"], Status: domain.StatusQueued}
		b, _ := json.Marshal(t)

		if _, err := s.C.Rdb.XAdd(ctx, &redis.XAddArgs{
			Stream: s.C.Cfg.StreamKey,
			Values: map[string]interface{}{"task": b},
		}).Result(); err == nil {
			_ = s.C.Rdb.ZRem(ctx, s.C.Cfg.ScheduledZSet, id).Err()
		} else {
			log.Ctx(ctx).Error().Err(err).Str("task_id", id).Msg("failed to XADD task")
		}
	}
	return nil
}

func fmtFloat(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }
