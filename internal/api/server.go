package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"redisq/internal/config"
	"redisq/internal/domain"
	"redisq/internal/infra/redisq"
	"redisq/internal/usecase"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

type enqueueReq struct {
	Type        string            `json:"type"`
	Payload     map[string]string `json:"payload"`
	MaxAttempts int               `json:"max_attempts"`
	RunAt       *int64            `json:"run_at_ms"` // optional delayed
}

func NewServer() *Server {
	ctx := context.Background()
	cfg := config.Load()

	cli := redisq.New(cfg.Redis)
	if err := cli.Init(ctx); err != nil {
		log.Ctx(ctx).Fatal().Msgf("something went wrong: %s", err)
	}

	enq := usecase.Enqueuer{Q: cli}
	r := chi.NewRouter()
	r.Post("/enqueue", func(w http.ResponseWriter, r *http.Request) {
		var req enqueueReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		t := domain.Task{Type: req.Type, Payload: req.Payload, MaxAttempts: req.MaxAttempts}

		var id string
		var err error

		if req.RunAt != nil {
			id, err = enq.At(r.Context(), t, time.UnixMilli(*req.RunAt))
		} else {
			id, err = enq.Now(r.Context(), t)
		}

		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": id})
	})

	return &Server{router: r}
}

type Server struct {
	router *chi.Mux
}

// Run method of the Server struct runs the HTTP server on the specified port. It initializes
// a new HTTP server instance with the specified port and the server's router.
func (s *Server) Run(port int) {
	addr := fmt.Sprintf(":%d", port)

	h := chainMiddleware(
		s.router,
		recoverHandler,
		loggerHandler(func(w http.ResponseWriter, r *http.Request) bool { return r.URL.Path == "/" }),
		realIPHandler,
		requestIDHandler,
		corsHandler,
	)

	httpServer := http.Server{
		Addr:         addr,
		Handler:      h,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-quit
		log.Info().Msg("Server is shutting down...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			log.Fatal().Err(err).Msg("Server forced to shutdown")
		}

		close(done)
	}()

	log.Info().Msgf("server serving on port %d", port)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("Failed to listen and serve")
	}

	<-done
	log.Info().Msg("Server stopped")
}
