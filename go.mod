module code.cloudfoundry.org/app-autoscaler/src/autoscaler

go 1.24.3

require (
	code.cloudfoundry.org/brokerapi/v13 v13.0.4
	code.cloudfoundry.org/cfhttp/v2 v2.45.0
	code.cloudfoundry.org/clock v1.38.0
	code.cloudfoundry.org/go-log-cache/v3 v3.1.1
	code.cloudfoundry.org/go-loggregator/v10 v10.2.0
	code.cloudfoundry.org/lager/v3 v3.37.0
	code.cloudfoundry.org/loggregator-agent-release/src v0.0.0-20250910195140-26c118c0f681
	code.cloudfoundry.org/tlsconfig v0.29.0
	github.com/cenkalti/backoff/v5 v5.0.2
	github.com/cloud-gov/go-cfenv v1.19.1
	github.com/go-chi/chi/v5 v5.2.2
	github.com/go-faster/errors v0.7.1
	github.com/go-faster/jx v1.1.0
	github.com/go-logr/logr v1.4.3
	github.com/go-sql-driver/mysql v1.9.2
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/hashicorp/go-retryablehttp v0.7.7
	github.com/jackc/pgx/v5 v5.7.6
	github.com/jmoiron/sqlx v1.4.0
	github.com/maxbrunsfeld/counterfeiter/v6 v6.11.2
	github.com/ogen-go/ogen v1.8.1
	github.com/onsi/ginkgo/v2 v2.23.4
	github.com/onsi/gomega v1.37.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/prometheus/client_golang v1.22.0
	github.com/rubyist/circuitbreaker v2.2.1+incompatible
	github.com/steinfletcher/apitest v1.6.0
	github.com/stretchr/testify v1.10.0
	github.com/tedsuo/ifrit v0.0.0-20230516164442-7862c310ad26
	github.com/uptrace/opentelemetry-go-extra/otelsql v0.3.2
	github.com/uptrace/opentelemetry-go-extra/otelsqlx v0.3.2
	github.com/xeipuuv/gojsonschema v1.2.0
	go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux v0.57.0
	go.opentelemetry.io/otel v1.36.0
	go.opentelemetry.io/otel/metric v1.36.0
	go.opentelemetry.io/otel/sdk v1.36.0
	go.opentelemetry.io/otel/trace v1.36.0
	go.yaml.in/yaml/v4 v4.0.0-rc.2
	golang.org/x/crypto v0.42.0
	golang.org/x/exp v0.0.0-20250911091902-df9299821621
	golang.org/x/net v0.44.0
	golang.org/x/time v0.12.0
	google.golang.org/grpc v1.74.2
)

replace google.golang.org/genproto => google.golang.org/genproto v0.0.0-20250922171735-9219d122eba9

require (
	code.cloudfoundry.org/go-diodes v0.0.0-20250505082646-e4c2d772c2ec // indirect
	code.cloudfoundry.org/go-metric-registry v0.0.0-20250611124144-ca953cadd16e // indirect
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/andybalholm/brotli v1.1.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cenk/backoff v2.2.1+incompatible // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dlclark/regexp2 v1.11.4 // indirect
	github.com/facebookgo/clock v0.0.0-20150410010913-600d898af40a // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.5.4 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-faster/yaml v0.4.6 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/pprof v0.0.0-20250607225305-033d6d78b36a // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.26.3 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/openzipkin/zipkin-go v0.4.3 // indirect
	github.com/peterbourgon/g2s v0.0.0-20170223122336-d4e7ad98afea // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.64.0 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	github.com/segmentio/asm v1.2.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.62.0 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.uber.org/automaxprocs v1.6.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/mod v0.28.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/sys v0.36.0 // indirect
	golang.org/x/text v0.29.0 // indirect
	golang.org/x/tools v0.37.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250908214217-97024824d090 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250908214217-97024824d090 // indirect
	google.golang.org/protobuf v1.36.9 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
