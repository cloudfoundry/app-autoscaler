package app

import (
	"net/http"
	"time"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/go-logr/zapr"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Router(logger *zap.Logger, timewaster TimeWaster, memoryTest MemoryGobbler,
	cpuTest CPUWaster, diskOccupier DiskOccupier, customMetricTest CustomMetricClient) *gin.Engine {
	r := gin.New()

	otel.SetTracerProvider(sdktrace.NewTracerProvider())
	otel.SetTextMapPropagator(propagation.TraceContext{})
	r.Use(otelgin.Middleware("acceptance-tests-go-app"))

	r.Use(ginzap.GinzapWithConfig(logger, &ginzap.Config{
		TimeFormat: time.RFC3339,
		UTC:        true,
		Context: ginzap.Fn(func(c *gin.Context) []zapcore.Field {
			fields := []zapcore.Field{}
			// log CF ID
			if requestID := c.Request.Header.Get("X-Vcap-Request-Id"); requestID != "" {
				fields = append(fields, zap.String("vcap_request_id", requestID))
			}
			if passport := c.Request.Header.Get("SAP-PASSPORT"); passport != "" {
				fields = append(fields, zap.String("sap_passport", passport))
			}
			// support opentelemetry trace ID
			fields = append(fields, zap.String("w3c_trace-id", trace.SpanFromContext(c.Request.Context()).SpanContext().TraceID().String()))

			return fields
		}),
	}))

	r.Use(ginzap.RecoveryWithZap(logger, true))

	logr := zapr.NewLogger(logger)

	r.GET("/", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"name": "test-app"}) })
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	MemoryTests(logr, r.Group("/memory"), memoryTest)
	ResponseTimeTests(logr, r.Group("/responsetime"), timewaster)
	CPUTests(logr, r.Group("/cpu"), cpuTest)
	DiskTest(r.Group("/disk"), diskOccupier)
	CustomMetricsTests(logr, r.Group("/custom-metrics"), customMetricTest)
	return r
}

func New(logger *zap.Logger, address string) *http.Server {
	errorLog, _ := zap.NewStdLogAt(logger, zapcore.ErrorLevel)
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
		ErrorLog:     errorLog,
	}
}
