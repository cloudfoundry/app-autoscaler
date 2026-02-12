package app

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// JSONResponse represents a JSON response structure
type JSONResponse map[string]interface{}

// writeJSON writes a JSON response with the given status code
func writeJSON(w http.ResponseWriter, statusCode int, data JSONResponse) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(data)
}

func Router(logger *slog.Logger, timewaster TimeWaster, memoryTest MemoryGobbler,
	cpuTest CPUWaster, diskOccupier DiskOccupier, customMetricTest CustomMetricClient) http.Handler {
	mux := http.NewServeMux()

	// Root routes

	// /{$} to match root path "/" exactly, see https://pkg.go.dev/net/http#hdr-Patterns-ServeMux
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		if err := writeJSON(w, http.StatusOK, JSONResponse{"name": "test-app"}); err != nil {
			logger.Error("Failed to write JSON response", slog.Any("error", err))
		}
	})

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		if err := writeJSON(w, http.StatusOK, JSONResponse{"status": "ok"}); err != nil {
			logger.Error("Failed to write JSON response", slog.Any("error", err))
		}
	})

	// Register test endpoints
	MemoryTests(logger, mux, memoryTest)
	ResponseTimeTests(logger, mux, timewaster)
	CPUTests(logger, mux, cpuTest)
	DiskTest(logger, mux, diskOccupier)
	CustomMetricsTests(logger, mux, customMetricTest)

	return loggingMiddleware(logger)(mux)
}

func New(logger *slog.Logger, address string) *http.Server {
	return &http.Server{
		Addr: address,
		Handler: Router(
			logger,
			&Sleeper{},
			&ListBasedMemoryGobbler{},
			&ConcurrentBusyLoopCPUWaster{},
			NewDefaultDiskOccupier("this-file-is-being-used-during-disk-occupation"),
			&CustomMetricAPIClient{},
		),
		ReadTimeout:  5 * time.Second,
		IdleTimeout:  2 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
}

func loggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap ResponseWriter to capture status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			attrs := []slog.Attr{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("user_agent", r.UserAgent()),
			}

			// Log Cloud Foundry (VCAP) request ID
			if requestID := r.Header.Get("X-Vcap-Request-Id"); requestID != "" {
				attrs = append(attrs, slog.String("vcap_request_id", requestID))
			}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)
			attrs = append(attrs,
				slog.Int("status_code", wrapped.statusCode),
				slog.Duration("duration", duration),
			)

			logger.LogAttrs(r.Context(), slog.LevelInfo, "HTTP request", attrs...)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code for logging
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
