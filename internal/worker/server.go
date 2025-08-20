// internal/worker/server.go
package worker

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"redisq/internal/config"
	"redisq/internal/domain"
	"redisq/internal/infra/redisq"
	"redisq/internal/usecase"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
)

type WorkerConfig struct {
	ConsumerName string
	BaseBackoff  time.Duration
	MaxBackoff   time.Duration
}

func Run(cfg WorkerConfig) error {
	appCfg := config.Load()
	cli := redisq.New(appCfg.Redis)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := cli.Init(ctx); err != nil {
		if !strings.Contains(err.Error(), "BUSYGROUP") {
			return err
		}
		log.Info().Msg("Consumer group already exists, continuing...")
	}

	// Run scheduler
	sched := redisq.NewScheduler(cli, 1*time.Second)
	go func() {
		if err := sched.Run(ctx); err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("scheduler stopped with error")
		}
	}()

	consumer := usecase.Consumer{
		Q:            cli,
		ConsumerName: cfg.ConsumerName,
		BaseBackoff:  cfg.BaseBackoff,
		MaxBackoff:   cfg.MaxBackoff,
	}

	handler := func(ctx context.Context, t domain.Task) error {
		if t.Type == "demo.fail" && t.Attempts < 2 {
			return errors.New("simulated failure")
		}
		log.Ctx(ctx).Info().Msgf("processed task %s type=%s attempts=%d", t.ID, t.Type, t.Attempts)
		return nil
	}

	log.Info().Msgf("Worker %s started. Waiting for tasks...", cfg.ConsumerName)
	return consumer.Run(ctx, handler)
}
