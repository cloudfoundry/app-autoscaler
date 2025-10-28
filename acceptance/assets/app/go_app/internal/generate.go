package internal

//go:generate go run github.com/ogen-go/ogen/cmd/ogen --config ogen-config.yaml --target applicationmetric --clean ../../../../../../api/application-metric-api.openapi.yaml
//go:generate go run github.com/ogen-go/ogen/cmd/ogen --config ogen-config.yaml --target custommetrics --clean ../../../../../../api/custom-metrics-api.openapi.yaml
//go:generate go run github.com/ogen-go/ogen/cmd/ogen --config ogen-config.yaml --target policy --clean ../../../../../../api/policy-api.openapi.yaml
