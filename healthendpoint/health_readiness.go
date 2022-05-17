package healthendpoint

import (
	"encoding/json"
	"net/http"
)

type (
	Pinger interface {
		Ping() error
	}

	ReadinessCheck struct {
		Name   string `json:"name"`
		Type   string `json:"type"`
		Status string `json:"status"`
	}
	readinessResponse struct {
		OverallStatus string           `json:"overall_status"`
		Checks        []ReadinessCheck `json:"checks"`
	}
	Checker func() ReadinessCheck
)

const (
	statusUp   = "UP"
	statusDown = "DOWN"
)

func readiness(checkers []Checker) func(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	return func(w http.ResponseWriter, r *http.Request, vars map[string]string) {
		w.Header().Set("Content-Type", "application/json")

		checks := make([]ReadinessCheck, 0, 8)
		overallStatus := statusUp
		for _, checker := range checkers {
			check := checker()
			checks = append(checks, check)
			if check.Status == statusDown {
				overallStatus = statusDown
			}
		}
		response, err := json.Marshal(readinessResponse{OverallStatus: overallStatus, Checks: checks})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"Internal error"}`))
		}
		_, _ = w.Write(response)
	}
}

func DbChecker(dbName string, pinger Pinger) Checker {
	if pinger != nil {
		return func() ReadinessCheck {
			status := statusUp
			err := pinger.Ping()
			if err != nil {
				status = statusDown
			}
			return ReadinessCheck{Name: dbName, Type: "database", Status: status}
		}
	} else {
		return func() ReadinessCheck {
			return ReadinessCheck{Name: dbName, Type: "database", Status: statusUp}
		}
	}
}
