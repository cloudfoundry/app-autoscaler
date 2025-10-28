package internal

//go:generate go run github.com/ogen-go/ogen/cmd/ogen --config ogen-config.yaml --target applicationmetric --clean ../../../../../openapi/application-metric-api.openapi.yaml
//go:generate go run github.com/ogen-go/ogen/cmd/ogen --config ogen-config.yaml --target custommetrics --clean ../../../../../openapi/custom-metrics-api.openapi.yaml
//go:generate go run github.com/ogen-go/ogen/cmd/ogen --config ogen-config.yaml --target policy --clean ../../../../../openapi/policy-api.openapi.yaml
