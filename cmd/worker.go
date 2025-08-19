package cmd

import (
	"redisq/internal/worker"
	"time"

	"github.com/spf13/cobra"
)

func workerCmd() *cobra.Command {
	var (
		consumerName string
		baseBackoff  time.Duration
		maxBackoff   time.Duration
	)

	var command = &cobra.Command{
		Use:   "worker",
		Short: "Start worker server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return worker.Run(worker.Config{
				ConsumerName: consumerName,
				BaseBackoff:  baseBackoff,
				MaxBackoff:   maxBackoff,
			})
		},
	}

	command.Flags().StringVar(&consumerName, "consumer", "worker-1", "Worker consumer name")
	command.Flags().DurationVar(&baseBackoff, "base-backoff", 500*time.Millisecond, "Base backoff duration")
	command.Flags().DurationVar(&maxBackoff, "max-backoff", 30*time.Second, "Max backoff duration")

	return command
}
