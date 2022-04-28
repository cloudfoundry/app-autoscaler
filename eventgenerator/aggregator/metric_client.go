package aggregator

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/config"
)

//{
//"envelopes": {
//"batch": [
//{
//"timestamp": "1651053182250846986",
//"source_id": "3fa04e83-bda3-4ff2-8bec-8f6b298b1384",
//"instance_id": "1",
//"deprecated_tags": {},
//"tags": {
//"app_id": "3fa04e83-bda3-4ff2-8bec-8f6b298b1384",
//"app_name": "autoscaler-1-nodeapp-c25d9f71ab95660d",
//"deployment": "cf_cells_1",
//"index": "3da5dc82-6a09-4016-b2c3-2f8b99518fe3",
//"instance_id": "1",
//"ip": "10.2.9.0",
//"job": "diego-cell",
//"organization_id": "e8c70374-c428-43f1-b42c-36b581ac6dab",
//"organization_name": "SAP_autoscaler_tests",
//"origin": "rep",
//"process_id": "3fa04e83-bda3-4ff2-8bec-8f6b298b1384",
//"process_instance_id": "3c5568ce-96ba-4dec-490e-207a",
//"process_type": "web",
//"source_id": "3fa04e83-bda3-4ff2-8bec-8f6b298b1384",
//"space_id": "6b79d201-1686-442f-9524-59ebf0ccfd88",
//"space_name": "acceptance_tests-1-SPACE-af380e0f940b8565"
//},
//"gauge": {
//"metrics": {
//"cpu": {
//"unit": "percentage",
//"value": 0.9534856922639902
//},
//"disk": {
//"unit": "bytes",
//"value": 118538240
//},
//"disk_quota": {
//"unit": "bytes",
//"value": 1073741824
//},
//"memory": {
//"unit": "bytes",
//"value": 44190393
//},
//"memory_quota": {
//"unit": "bytes",
//"value": 134217728
//}
//}
//}
//},
type MetricClient struct {
	config config.Config
}

type LogCacheResponse struct {
}

func (c *MetricClient) GetAddrs() string {
	return c.config.MetricCollector.MetricCollectorURL
}

func (c *MetricClient) EnableLogCache() bool {
	return c.config.UseLogCache
}

func NewMetricClient(config config.Config) MetricClient {
	return MetricClient{config: config}
}
