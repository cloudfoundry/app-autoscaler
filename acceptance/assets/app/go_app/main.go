package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"code.cloudfoundry.org/app-autoscaler-release/src/acceptance/assets/app/go_app/internal/app"
)

func main() {
	logger := createLogger()

	address := os.Getenv("SERVER_ADDRESS") + ":" + getPort(logger)
	logger.Info("Starting test-app", slog.String("address", address))
	server := app.New(logger, address)
	enableGracefulShutdown(logger, server)
	err := server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("Error while exiting server", slog.Any("error", err))
		os.Exit(1)
	}
}

func createLogger() *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	handler := slog.NewTextHandler(os.Stdout, opts)
	logger := slog.New(handler)

	return logger
}

func enableGracefulShutdown(logger *slog.Logger, server *http.Server) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		// When we get the signal...
		<-stop
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		// ... we gracefully shut down the server.
		// That ensures that no new connections
		err := server.Shutdown(ctx)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Error closing server", slog.Any("error", err))
			err = server.Close()
			if err != nil {
				logger.Error("Error while forcefully closing", slog.Any("error", err))
			}
		}
	}()
}

func getPort(logger *slog.Logger) string {
	port := os.Getenv("PORT")
	if port == "" {
		logger.Info("No Env var PORT specified using 8080")
		port = "8080"
	}
	return port
}
