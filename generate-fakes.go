package fakes

// Multiple go:generate directives instead of counterfeiter:generate due to https://github.com/maxbrunsfeld/counterfeiter/issues/254
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o ./fakes/fake_cf_client.go ./cf CFClient
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o ./fakes/fake_cf_context_client.go ./cf ContextClient
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o ./fakes/fake_policy_db.go ./db PolicyDB
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o ./fakes/fake_scalingengine_db.go ./db ScalingEngineDB
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o ./fakes/fake_scheduler_db.go ./db SchedulerDB
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o ./fakes/fake_scalingengine.go ./scalingengine ScalingEngine
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o ./fakes/fake_binding_db.go ./db BindingDB
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o ./fakes/fake_app_metric_db.go ./db AppMetricDB
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o ./fakes/fake_credentials.go ./cred_helper Credentials
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o ./fakes/fake_storedprocedure_db.go ./db StoredProcedureDB
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o ./fakes/fake_metric_forwarder.go ./metricsforwarder/forwarder MetricForwarder
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o ./fakes/fake_plan_checker.go ./api/plancheck PlanChecker
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o ./fakes/fake_log_cache_client.go ./eventgenerator/client LogCacheClientReader
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o ./fakes/fake_envelope_processor.go ./envelopeprocessor EnvelopeProcessor
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o ./fakes/fake_log_cache_creator.go ./eventgenerator/client LogCacheClientCreator
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o ./fakes/fake_go_log_cache_client.go ./eventgenerator/client GoLogCacheClient
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o ./fakes/fake_grpc.go ./eventgenerator/client GRPCOptions
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o ./fakes/fake_envelope_processor_creator.go ./envelopeprocessor EnvelopeProcessorCreator
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o ./fakes/fake_plan_checker.go ./api/plancheck PlanChecker
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o ./fakes/fake_ratelimiter.go ./ratelimiter Limiter
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o ./fakes/fake_httpstatus_collector.go ./healthendpoint HTTPStatusCollector
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o ./fakes/fake_database_status.go ./healthendpoint DatabaseStatus
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o ./fakes/fake_operator.go ./operator Operator
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o ./fakes/fake_sychronizer.go ./scalingengine/schedule ActiveScheduleSychronizer
