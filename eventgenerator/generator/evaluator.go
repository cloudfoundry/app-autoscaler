package generator

import (
	"autoscaler/db"
	"autoscaler/models"
	"autoscaler/routes"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/rubyist/circuitbreaker"
)

var validOperators = []string{">", ">=", "<", "<="}

type Evaluator struct {
	logger                    lager.Logger
	httpClient                *http.Client
	scalingEngineUrl          string
	triggerChan               chan []*models.Trigger
	doneChan                  chan bool
	database                  db.AppMetricDB
	defaultBreachDurationSecs int
	getBreaker                func(string) *circuit.Breaker
}

func NewEvaluator(logger lager.Logger, httpClient *http.Client, scalingEngineUrl string, triggerChan chan []*models.Trigger,
	database db.AppMetricDB, defaultBreachDurationSecs int, getBreaker func(string) *circuit.Breaker) *Evaluator {
	return &Evaluator{
		logger:                    logger.Session("Evaluator"),
		httpClient:                httpClient,
		scalingEngineUrl:          scalingEngineUrl,
		triggerChan:               triggerChan,
		doneChan:                  make(chan bool),
		database:                  database,
		defaultBreachDurationSecs: defaultBreachDurationSecs,
		getBreaker:                getBreaker,
	}
}

func (e *Evaluator) Start() {
	go e.start()
	e.logger.Info("started")
}

func (e *Evaluator) start() {
	for {
		select {
		case <-e.doneChan:
			return
		case triggerArray := <-e.triggerChan:
			e.doEvaluate(triggerArray)
		}
	}
}

func (e *Evaluator) Stop() {
	e.doneChan <- true
	close(e.doneChan)
	e.logger.Info("stopped")
}

func (e *Evaluator) doEvaluate(triggerArray []*models.Trigger) {
	for _, trigger := range triggerArray {
		threshold := trigger.Threshold
		operator := trigger.Operator

		if !e.isValidOperator(operator) {
			e.logger.Error("operator is invalid", nil, lager.Data{"trigger": trigger})
			continue
		}

		appMetricList, err := e.retrieveAppMetrics(trigger)
		if err != nil {
			continue
		}
		if len(appMetricList) == 0 {
			e.logger.Debug("no available appmetric", lager.Data{"trigger": trigger})
			continue
		}

		isBreached := true
		for _, appMetric := range appMetricList {
			if appMetric.Value == "" {
				e.logger.Debug("should not send trigger alarm to scaling engine because there is empty value metric", lager.Data{"trigger": trigger, "appMetric": appMetric})
				isBreached = false
				break
			}
			value, err := strconv.ParseInt(appMetric.Value, 10, 64)
			if err != nil {
				e.logger.Debug("should not send trigger alarm to scaling engine because parse metric value fails", lager.Data{"trigger": trigger, "appMetric": appMetric})
				isBreached = false
				break
			}

			if operator == ">" {
				if value <= threshold {
					e.logger.Debug("should not send trigger alarm to scaling engine", lager.Data{"trigger": trigger, "appMetric": appMetric})
					isBreached = false
					break
				}
			} else if operator == ">=" {
				if value < threshold {
					e.logger.Debug("should not send trigger alarm to scaling engine", lager.Data{"trigger": trigger, "appMetric": appMetric})
					isBreached = false
					break
				}
			} else if operator == "<" {
				if value >= threshold {
					e.logger.Debug("should not send trigger alarm to scaling engine", lager.Data{"trigger": trigger, "appMetric": appMetric})
					isBreached = false
					break
				}
			} else if operator == "<=" {
				if value > threshold {
					e.logger.Debug("should not send trigger alarm to scaling engine", lager.Data{"trigger": trigger, "appMetric": appMetric})
					isBreached = false
					break
				}
			}
		}

		if isBreached {
			trigger.MetricUnit = appMetricList[0].Unit
			e.logger.Info("send trigger alarm to scaling engine", lager.Data{"trigger": trigger})

			if appBreaker := e.getBreaker(trigger.AppId); appBreaker != nil {
				appBreaker.Call(func() error {
					return e.sendTriggerAlarm(trigger)
				}, 0)
			} else {
				e.sendTriggerAlarm(trigger)
			}
			return
		}
	}

}

func (e *Evaluator) retrieveAppMetrics(trigger *models.Trigger) ([]*models.AppMetric, error) {
	queryEndTime := time.Now()
	queryStartTime := queryEndTime.Add(0 - 2*trigger.BreachDuration(e.defaultBreachDurationSecs))
	breachStartTime := queryEndTime.Add(0 - trigger.BreachDuration(e.defaultBreachDurationSecs))
	appMetrics, err := e.database.RetrieveAppMetrics(trigger.AppId, trigger.MetricType, queryStartTime.UnixNano(), queryEndTime.UnixNano())
	if err != nil {
		e.logger.Error("retrieve appMetrics", err, lager.Data{"trigger": trigger})
		return nil, err
	}
	e.logger.Debug("appMetrics", lager.Data{"appMetrics": appMetrics})
	result := []*models.AppMetric{}
	if len(appMetrics) > 0 {
		if appMetrics[0].Timestamp < breachStartTime.UnixNano() {
			for i := len(appMetrics) - 1; i >= 0; i-- {
				if appMetrics[i].Timestamp >= breachStartTime.UnixNano() {
					result = append(result, appMetrics[i])
				} else {
					break
				}
			}
		} else {
			e.logger.Debug("the appmetrics are not enough for evaluation", lager.Data{"trigger": trigger, "appMetrics": appMetrics})
		}
	}
	return result, nil
}

func (e *Evaluator) sendTriggerAlarm(trigger *models.Trigger) error {
	jsonBytes, err := json.Marshal(trigger)
	if err != nil {
		e.logger.Error("failed-marshal-trigger", err)
		return nil
	}

	path, _ := routes.ScalingEngineRoutes().Get(routes.ScaleRouteName).URLPath("appid", trigger.AppId)
	resp, err := e.httpClient.Post(e.scalingEngineUrl+path.Path, "application/json", bytes.NewReader(jsonBytes))
	if err != nil {
		e.logger.Error("failed-send-trigger-alarm-request", err, lager.Data{"trigger": trigger})
		return err
	}

	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		e.logger.Error("failed-read-response-body-from-scaling-engine", err)
	}

	if resp.StatusCode == http.StatusOK {
		e.logger.Info("successfully-send-trigger-alarm", lager.Data{"trigger": trigger})
		return nil
	}
	err = fmt.Errorf("Got %d when sending trigger alarm", resp.StatusCode)
	e.logger.Error("failed-send-trigger-alarm", err, lager.Data{"trigger": trigger, "responseBody": respBody})
	return err

}
func (e *Evaluator) isValidOperator(operator string) bool {
	for _, o := range validOperators {
		if o == operator {
			return true
		}
	}
	return false
}
