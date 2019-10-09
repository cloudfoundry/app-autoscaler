package fakes

//go:generate counterfeiter -o ./fake_cf_client.go ../cf CFClient
//go:generate counterfeiter -o ./fake_policy_db.go ../db PolicyDB
//go:generate counterfeiter -o ./fake_scalingengine_db.go ../db ScalingEngineDB
//go:generate counterfeiter -o ./fake_scheduler_db.go ../db SchedulerDB
//go:generate counterfeiter -o ./fake_scalingengine.go ../scalingengine ScalingEngine
//go:generate counterfeiter -o ./fake_binding_db.go ../db BindingDB
