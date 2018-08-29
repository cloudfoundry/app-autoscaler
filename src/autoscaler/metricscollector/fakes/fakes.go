package fakes

//go:generate counterfeiter -o ./fake_cf_client.go ../../cf CFClient
//go:generate counterfeiter -o ./fake_noaa_consumer.go ../noaa NoaaConsumer
//go:generate counterfeiter -o ./fake_policy_db.go ../../db PolicyDB
//go:generate counterfeiter -o ./fake_instancemetrics_db.go ../../db InstanceMetricsDB
//go:generate counterfeiter -o ./fake_appmetrics_db.go ../../db AppMetricDB
//go:generate counterfeiter -o ./fake_app_collector.go ../collector AppCollector
