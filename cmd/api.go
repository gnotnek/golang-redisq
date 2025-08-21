package cmd

import (
	"redisq/internal/api"
	"redisq/internal/config"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func apiCmd() *cobra.Command {
	var port int
	var command = &cobra.Command{
		Use:   "api",
		Short: "Start API server",
		Run: func(cmd *cobra.Command, args []string) {
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
			cfg := config.Load()
			log.Info().Msgf("API server using stream: %s, group: %s", cfg.Redis.StreamKey, cfg.Redis.Group)
			server := api.NewServer()
			server.Run(port)
		},
	}

	command.Flags().IntVarP(&port, "port", "p", 8080, "Port to run the server on")
	return command
}
