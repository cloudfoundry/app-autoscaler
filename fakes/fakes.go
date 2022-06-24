package fakes

//go:generate counterfeiter -o ./fake_cf_client.go ../cf CFClient
//go:generate counterfeiter -o ./fake_policy_db.go ../db PolicyDB
//go:generate counterfeiter -o ./fake_scalingengine_db.go ../db ScalingEngineDB
//go:generate counterfeiter -o ./fake_scheduler_db.go ../db SchedulerDB
//go:generate counterfeiter -o ./fake_scalingengine.go ../scalingengine ScalingEngine
//go:generate counterfeiter -o ./fake_binding_db.go ../db BindingDB
//go:generate counterfeiter -o ./fake_credentials.go ../cred_helper Credentials
//go:generate counterfeiter -o ./fake_storedprocedure_db.go ../db StoredProcedureDB
//go:generate counterfeiter -o ./fake_database_status.go ../healthendpoint DatabaseStatus
//go:generate counterfeiter -o ./fake_log_cache_client.go ../eventgenerator/aggregator LogCacheClientReader
//go:generate counterfeiter -o ./fake_envelope_processor.go ../envelopeprocessor EnvelopeProcessor
