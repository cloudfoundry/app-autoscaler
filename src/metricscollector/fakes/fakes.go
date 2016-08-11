package fakes

//go:generate counterfeiter -o ./fake_cf_client.go ../cf CfClient
//go:generate counterfeiter -o ./fake_noaa_consumer.go ../noaa NoaaConsumer
//go:generate counterfeiter -o ./fake_DB.go ../db DB
//go:generate counterfeiter -o ./fake_app_poller.go ../collector AppPoller
