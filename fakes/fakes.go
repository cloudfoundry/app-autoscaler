package fakes

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate -o ./fake_wshelper.go ../metricsgateway/helpers WSHelper
//counterfeiter:generate -o ./fake_cf_client.go ../cf CFClient
//counterfeiter:generate -o ./fake_policy_db.go ../db PolicyDB
//counterfeiter:generate -o ./fake_scalingengine_db.go ../db ScalingEngineDB
//counterfeiter:generate -o ./fake_scheduler_db.go ../db SchedulerDB
//counterfeiter:generate -o ./fake_scalingengine.go ../scalingengine ScalingEngine
//counterfeiter:generate -o ./fake_binding_db.go ../db BindingDB
//counterfeiter:generate -o ./fake_app_metric_db.go ../db AppMetricDB
//counterfeiter:generate -o ./fake_instancemetrics_db.go ../db InstanceMetricsDB
//counterfeiter:generate -o ./fake_credentials.go ../cred_helper Credentials
//counterfeiter:generate -o ./fake_storedprocedure_db.go ../db StoredProcedureDB
//counterfeiter:generate -o ./fake_metric_forwarder.go ../metricsforwarder/forwarder MetricForwarder
//counterfeiter:generate -o ./fake_plan_checker.go ../api/plancheck PlanChecker
//counterfeiter:generate -o ./fake_log_cache_client.go ../eventgenerator/client LogCacheClientReader
//counterfeiter:generate -o ./fake_envelope_processor.go ../envelopeprocessor EnvelopeProcessor
//counterfeiter:generate -o ./fake_log_cache_creator.go ../eventgenerator/client LogCacheClientCreator
//counterfeiter:generate -o ./fake_metric_server_creator_creator.go ../eventgenerator/client MetricServerClientCreator
//counterfeiter:generate -o ./fake_go_log_cache_client.go ../eventgenerator/client GoLogCacheClient
//counterfeiter:generate -o ./fake_grpc.go ../eventgenerator/client GRPCOptions
//counterfeiter:generate -o ./fake_envelope_processor_creator.go ../envelopeprocessor EnvelopeProcessorCreator
//counterfeiter:generate -o ./fake_plan_checker.go ../api/plancheck PlanChecker
//counterfeiter:generate -o ./fake_ratelimiter.go ../ratelimiter Limiter
//counterfeiter:generate -o ./fake_httpstatus_collector.go ../healthendpoint HTTPStatusCollector
//counterfeiter:generate -o ./fake_database_status.go ../healthendpoint DatabaseStatus
//counterfeiter:generate -o ./fake_emitter.go ../metricsgateway Emitter
//counterfeiter:generate -o ./fake_operator.go ../operator Operator
//counterfeiter:generate -o ./fake_sychronizer.go ../scalingengine/schedule ActiveScheduleSychronizer
