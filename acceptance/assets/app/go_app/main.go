package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"code.cloudfoundry.org/app-autoscaler-release/src/acceptance/assets/app/go_app/internal/app"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	logger := createLogger()
	sugar := logger.Sugar()

	gin.SetMode(gin.ReleaseMode)

	address := os.Getenv("SERVER_ADDRESS") + ":" + getPort(sugar)
	sugar.Infof("Starting test-app : %s\n", address)
	server := app.New(logger, address)
	enableGracefulShutdown(sugar, server)
	err := server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		sugar.Panicf("Error while exiting server: %s", err.Error())
	}
}

func createLogger() *zap.Logger {
	highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.ErrorLevel
	})
	lowPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl < zapcore.ErrorLevel
	})

	consoleDebugging := zapcore.Lock(os.Stdout)
	consoleErrors := zapcore.Lock(os.Stderr)

	consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())

	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, consoleErrors, highPriority),
		zapcore.NewCore(consoleEncoder, consoleDebugging, lowPriority),
	)

	// From a zapcore.Core, it's easy to construct a Logger.
	logger := zap.New(core)

	return logger
}

func enableGracefulShutdown(logger *zap.SugaredLogger, server *http.Server) {
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
			logger.Errorf("Error closing server: %s", err.Error())
			err = server.Close()
			if err != nil {
				logger.Errorf("Error while forcefully closing: %s", err.Error())
			}
		}
	}()
}

func getPort(logger *zap.SugaredLogger) string {
	port := os.Getenv("PORT")
	if port == "" {
		logger.Infof("No Env var PORT specified using 8080")
		port = "8080"
	}
	return port
}
