package redisq

import (
	"context"
	"encoding/json"
	"fmt"
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
		if err := s.moveDue(ctx); err != nil {
			/* log */
			log.Ctx(ctx).Err(err).Msgf("something went wrong: %s", err)
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
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

	if err != nil || len(ids) == 0 {
		return fmt.Errorf("something happen")
	}

	for _, id := range ids {
		key := "task:" + id
		h, _ := s.C.Rdb.HGetAll(ctx, key).Result()

		t := domain.Task{ID: id, Type: h["type"], Status: domain.StatusQueued}
		b, _ := json.Marshal(t)

		if _, err := s.C.Rdb.XAdd(ctx, &redis.XAddArgs{
			Stream: s.C.Cfg.StreamKey,
			Values: map[string]interface{}{"task": b}}).Result(); err == nil {
			_ = s.C.Rdb.ZRem(ctx, s.C.Cfg.ScheduledZSet, id).Err()
		}
	}
	return nil
}

func fmtFloat(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }
