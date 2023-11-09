module code.cloudfoundry.org/app-autoscaler/src/autoscaler

go 1.20

require (
	code.cloudfoundry.org/cfhttp/v2 v2.0.1-0.20230113212937-05beac96f8c7
	code.cloudfoundry.org/clock v1.1.0
	code.cloudfoundry.org/go-log-cache/v2 v2.0.5
	code.cloudfoundry.org/go-loggregator/v9 v9.0.8
	code.cloudfoundry.org/lager/v3 v3.0.2
	code.cloudfoundry.org/tlsconfig v0.0.0-20231017135636-f0e44068c22f
	dario.cat/mergo v1.0.0
	github.com/cenkalti/backoff/v4 v4.2.1
	github.com/go-chi/chi/v5 v5.0.10
	github.com/go-faster/errors v0.7.0
	github.com/go-faster/jx v1.1.0
	github.com/go-sql-driver/mysql v1.7.1
	github.com/golang/protobuf v1.5.3
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/websocket v1.5.1
	github.com/hashicorp/go-retryablehttp v0.7.4
	github.com/jackc/pgx/v5 v5.5.0
	github.com/jmoiron/sqlx v1.3.5
	github.com/maxbrunsfeld/counterfeiter/v6 v6.7.0
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d
	github.com/ogen-go/ogen v0.77.0
	github.com/onsi/ginkgo/v2 v2.13.0
	github.com/onsi/gomega v1.29.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pivotal-cf/brokerapi/v10 v10.1.1
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.17.0
	github.com/rubyist/circuitbreaker v2.2.1+incompatible
	github.com/steinfletcher/apitest v1.5.15
	github.com/stretchr/testify v1.8.4
	github.com/tedsuo/ifrit v0.0.0-20230516164442-7862c310ad26
	github.com/xeipuuv/gojsonschema v1.2.0
	go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux v0.45.0
	go.opentelemetry.io/otel v1.19.0
	go.opentelemetry.io/otel/metric v1.19.0
	go.opentelemetry.io/otel/sdk v1.19.0
	go.opentelemetry.io/otel/trace v1.19.0
	golang.org/x/crypto v0.15.0
	golang.org/x/exp v0.0.0-20231006140011-7918f672742d
	golang.org/x/net v0.17.0
	golang.org/x/time v0.4.0
	google.golang.org/grpc v1.59.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	code.cloudfoundry.org/go-diodes v0.0.0-20221212174934-b8cb650f2489 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cenk/backoff v2.2.1+incompatible // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dlclark/regexp2 v1.10.0 // indirect
	github.com/facebookgo/clock v0.0.0-20150410010913-600d898af40a // indirect
	github.com/fatih/color v1.15.0 // indirect
	github.com/felixge/httpsnoop v1.0.3 // indirect
	github.com/fsnotify/fsnotify v1.5.4 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-faster/yaml v0.4.6 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/pprof v0.0.0-20230228050547-1710fef4ab10 // indirect
	github.com/google/uuid v1.4.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.18.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v1.2.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/openzipkin/zipkin-go v0.4.1 // indirect
	github.com/pborman/uuid v1.2.1 // indirect
	github.com/peterbourgon/g2s v0.0.0-20170223122336-d4e7ad98afea // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.4.1-0.20230718164431-9a2bf3000d16 // indirect
	github.com/prometheus/common v0.44.0 // indirect
	github.com/prometheus/procfs v0.11.1 // indirect
	github.com/segmentio/asm v1.2.0 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20180127040702-4e3ac2762d5f // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.26.0 // indirect
	golang.org/x/mod v0.13.0 // indirect
	golang.org/x/sync v0.4.0 // indirect
	golang.org/x/sys v0.14.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/tools v0.14.0 // indirect
	google.golang.org/genproto v0.0.0-20230822172742-b8732ec3820d // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20230822172742-b8732ec3820d // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230822172742-b8732ec3820d // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)
