package cmd

import (
	"redisq/internal/api"

	"github.com/spf13/cobra"
)

func apiCmd() *cobra.Command {
	var port int
	var command = &cobra.Command{
		Use:   "api",
		Short: "Start API server",
		Run: func(cmd *cobra.Command, args []string) {
			server := api.NewServer()
			server.Run(port)
		},
	}

	command.Flags().IntVarP(&port, "port", "p", 8080, "Port to run the server on")
	return command
}
