package redisq

import (
	"context"
	"fmt"
	"redisq/internal/config"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type Client struct {
	Cfg config.Redis
	Rdb *redis.Client
}

func New(cfg config.Redis) *Client {
	log.Info().Msgf("connecting to redis at %s", cfg.Addr)
	c := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	return &Client{Cfg: cfg, Rdb: c}
}

// Connect → used by API only
func (c *Client) Connect(ctx context.Context) error {
	if err := c.Rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis connection failed: %w", err)
	}
	log.Ctx(ctx).Info().Msg("connected to redis")
	return nil
}

// Init → used by Worker, ensures stream + group exist
func (c *Client) Init(ctx context.Context) error {
	if err := c.Connect(ctx); err != nil {
		return err
	}

	// Create stream and group if not exists
	err := c.Rdb.XGroupCreateMkStream(ctx, c.Cfg.StreamKey, c.Cfg.Group, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP Consumer Group name already exists") {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("stream", c.Cfg.StreamKey).
		Str("group", c.Cfg.Group).
		Msg("redis stream and consumer group ready")

	return nil
}

func nowMs() float64 { return float64(time.Now().UnixMilli()) }
