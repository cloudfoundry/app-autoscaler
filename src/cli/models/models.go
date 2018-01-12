package models

import "autoscaler/models"

type MetricsResults struct {
	TotalResults uint32                      `json:"total_results"`
	TotalPages   uint16                      `json:"total_pages"`
	Page         uint16                      `json:"page"`
	Metrics      []*models.AppInstanceMetric `json:"resources"`
}

type HistoryResults struct {
	TotalResults uint32                      `json:"total_results"`
	TotalPages   uint16                      `json:"total_pages"`
	Page         uint16                      `json:"page"`
	Histories    []*models.AppScalingHistory `json:"resources"`
}
