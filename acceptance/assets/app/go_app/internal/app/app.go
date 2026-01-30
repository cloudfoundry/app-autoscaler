package app

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// JSONResponse represents a JSON response structure
type JSONResponse map[string]interface{}

// writeJSON writes a JSON response with the given status code
func writeJSON(w http.ResponseWriter, statusCode int, data JSONResponse) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(data)
}

// loggingMiddleware provides request logging functionality
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

			// Log CF ID
			if requestID := r.Header.Get("X-Vcap-Request-Id"); requestID != "" {
				attrs = append(attrs, slog.String("vcap_request_id", requestID))
			}
			if passport := r.Header.Get("SAP-PASSPORT"); passport != "" {
				attrs = append(attrs, slog.String("sap_passport", passport))
			}

			// Support OpenTelemetry trace ID
			if span := trace.SpanFromContext(r.Context()); span.SpanContext().IsValid() {
				attrs = append(attrs, slog.String("w3c_trace-id", span.SpanContext().TraceID().String()))
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

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// recoveryMiddleware provides panic recovery functionality
func recoveryMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("Panic recovered",
						slog.Any("error", err),
						slog.String("stack", string(debug.Stack())),
						slog.String("path", r.URL.Path),
						slog.String("method", r.Method),
					)
					http.Error(w, "Internal Server Errorf", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// otelMiddleware provides OpenTelemetry tracing
func otelMiddleware(next http.Handler) http.Handler {
	tracer := otel.Tracer("acceptance-tests-go-app")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		ctx, span := tracer.Start(ctx, fmt.Sprintf("%s %s", r.Method, r.URL.Path))
		defer span.End()

		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

func Router(logger *slog.Logger, timewaster TimeWaster, memoryTest MemoryGobbler,
	cpuTest CPUWaster, diskOccupier DiskOccupier, customMetricTest CustomMetricClient) http.Handler {
	mux := http.NewServeMux()

	// Set up OpenTelemetry
	otel.SetTracerProvider(sdktrace.NewTracerProvider())
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Root routes
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
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

	// Apply middleware in order: recovery -> logging -> otel -> router
	var handler http.Handler = mux
	handler = otelMiddleware(handler)
	handler = loggingMiddleware(logger)(handler)
	handler = recoveryMiddleware(logger)(handler)

	return handler
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
