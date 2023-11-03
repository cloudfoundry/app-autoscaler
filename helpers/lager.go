package helpers

import (
	"context"

	"code.cloudfoundry.org/lager/v3"
	"go.opentelemetry.io/otel/trace"
)

func AddTraceID(ctx context.Context, data lager.Data) lager.Data {
	traceId := trace.SpanFromContext(ctx).SpanContext().TraceID().String()
	data["w3c_trace-id"] = traceId
	return data
}
