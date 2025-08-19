package redisq

import (
	"context"
	"redisq/internal/config"
	"time"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	Cfg config.Redis
	Rdb *redis.Client
}

func New(cfg config.Redis) *Client {
	c := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	return &Client{Cfg: cfg, Rdb: c}
}

func (c *Client) Init(ctx context.Context) error {
	// Create consumer group idempotently
	_, err := c.Rdb.XGroupCreateMkStream(ctx, c.Cfg.StreamKey, c.Cfg.Group,
		"$").Result()
	if err != nil && !isGroupExists(err) {
		return err
	}
	return nil
}

func isGroupExists(err error) bool {
	return err != nil && (err.Error() == "BUSYGROUP Consumer Group namealready exists" ||
		err.Error() == "BUSYGROUP Consumer Group namealready exists for this key")
}

func nowMs() float64 { return float64(time.Now().UnixMilli()) }
