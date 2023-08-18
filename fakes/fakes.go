package fakes

// Multiple go:generate directives instead of counterfeiter:generate due to https://github.com/maxbrunsfeld/counterfeiter/issues/254
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6  -o ./fake_wshelper.go ../metricsgateway/helpers WSHelper
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6  -o ./fake_cf_client.go ../cf CFClient
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6  -o ./fake_policy_db.go ../db PolicyDB
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6  -o ./fake_scalingengine_db.go ../db ScalingEngineDB
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6  -o ./fake_scheduler_db.go ../db SchedulerDB
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6  -o ./fake_scalingengine.go ../scalingengine ScalingEngine
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6  -o ./fake_binding_db.go ../db BindingDB
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6  -o ./fake_app_metric_db.go ../db AppMetricDB
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6  -o ./fake_instancemetrics_db.go ../db InstanceMetricsDB
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6  -o ./fake_credentials.go ../cred_helper Credentials
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6  -o ./fake_storedprocedure_db.go ../db StoredProcedureDB
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6  -o ./fake_metric_forwarder.go ../metricsforwarder/forwarder MetricForwarder
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6  -o ./fake_plan_checker.go ../api/plancheck PlanChecker
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6  -o ./fake_log_cache_client.go ../eventgenerator/client LogCacheClientReader
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6  -o ./fake_envelope_processor.go ../envelopeprocessor EnvelopeProcessor
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6  -o ./fake_log_cache_creator.go ../eventgenerator/client LogCacheClientCreator
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6  -o ./fake_metric_server_creator_creator.go ../eventgenerator/client MetricServerClientCreator
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6  -o ./fake_go_log_cache_client.go ../eventgenerator/client GoLogCacheClient
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6  -o ./fake_grpc.go ../eventgenerator/client GRPCOptions
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6  -o ./fake_envelope_processor_creator.go ../envelopeprocessor EnvelopeProcessorCreator
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6  -o ./fake_plan_checker.go ../api/plancheck PlanChecker
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6  -o ./fake_ratelimiter.go ../ratelimiter Limiter
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6  -o ./fake_httpstatus_collector.go ../healthendpoint HTTPStatusCollector
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6  -o ./fake_database_status.go ../healthendpoint DatabaseStatus
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6  -o ./fake_emitter.go ../metricsgateway Emitter
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6  -o ./fake_operator.go ../operator Operator
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6  -o ./fake_sychronizer.go ../scalingengine/schedule ActiveScheduleSychronizer
