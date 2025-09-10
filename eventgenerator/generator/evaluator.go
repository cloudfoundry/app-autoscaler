package generator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/aggregator"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"

	"code.cloudfoundry.org/lager/v3"
	circuit "github.com/rubyist/circuitbreaker"
)

var validOperators = []string{">", ">=", "<", "<="}

type Evaluator struct {
	logger                    lager.Logger
	httpClient                *http.Client
	scalingEngineUrl          string
	triggerChan               chan []*models.Trigger
	doneChan                  chan bool
	defaultBreachDurationSecs int
	queryAppMetrics           aggregator.QueryAppMetricsFunc
	getBreaker                func(string) *circuit.Breaker
	setCoolDownExpired        func(string, int64)
}

func NewEvaluator(logger lager.Logger, httpClient *http.Client, scalingEngineUrl string, triggerChan chan []*models.Trigger,
	defaultBreachDurationSecs int, queryAppMetrics aggregator.QueryAppMetricsFunc, getBreaker func(string) *circuit.Breaker, setCoolDownExpired func(string, int64)) *Evaluator {
	return &Evaluator{
		logger:                    logger.Session("Evaluator"),
		httpClient:                httpClient,
		scalingEngineUrl:          scalingEngineUrl,
		triggerChan:               triggerChan,
		doneChan:                  make(chan bool),
		defaultBreachDurationSecs: defaultBreachDurationSecs,
		queryAppMetrics:           queryAppMetrics,
		getBreaker:                getBreaker,
		setCoolDownExpired:        setCoolDownExpired,
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
		if trigger.BreachDurationSeconds <= 0 {
			trigger.BreachDurationSeconds = e.defaultBreachDurationSecs
		}
		threshold := trigger.Threshold
		operator := trigger.Operator
		if !e.isValidOperator(operator) {
			e.logger.Error("operator-is-invalid", nil, lager.Data{"trigger": trigger})
			continue
		}

		appMetricList, err := e.retrieveAppMetrics(trigger)
		if err != nil {
			continue
		}
		if len(appMetricList) == 0 {
			e.logger.Debug("no-available-appmetric", lager.Data{"trigger": trigger})
			continue
		}

		isBreached, appMetric := checkForBreach(appMetricList, e, trigger, operator, threshold)

		if isBreached {
			trigger.MetricUnit = appMetricList[0].Unit
			e.logger.Info("send trigger alarm to scaling engine", lager.Data{"trigger": trigger, "last_metric": appMetric})

			if appBreaker := e.getBreaker(trigger.AppId); appBreaker != nil {
				if appBreaker.Tripped() {
					e.logger.Info("circuit-tripped", lager.Data{"appId": trigger.AppId, "consecutiveFailures": appBreaker.ConsecFailures()})
				}
				err = appBreaker.Call(func() error { return e.sendTriggerAlarm(trigger) }, 0)
				if err != nil {
					e.logger.Error("circuit-alarm-failed", err, lager.Data{"appId": trigger.AppId})
				}
			} else {
				err = e.sendTriggerAlarm(trigger)
				if err != nil {
					e.logger.Error("circuit-alarm-failed", err, lager.Data{"appId": trigger.AppId})
				}
			}
			return
		}
	}
}

func checkForBreach(appMetricList []*models.AppMetric, e *Evaluator, trigger *models.Trigger, operator string, threshold int64) (bool, *models.AppMetric) {
	var appMetric *models.AppMetric
	for _, appMetric = range appMetricList {
		if appMetric.Value == "" {
			e.logger.Debug("should not send trigger alarm to scaling engine because there is empty value metric", lager.Data{"trigger": trigger, "appMetric": appMetric})
			return false, appMetric
		}
		value, err := strconv.ParseInt(appMetric.Value, 10, 64)
		if err != nil {
			e.logger.Debug("should not send trigger alarm to scaling engine because parse metric value fails", lager.Data{"trigger": trigger, "appMetric": appMetric})
			return false, appMetric
		}
		switch operator {
		case ">":
			if value <= threshold {
				e.logger.Debug("should not send trigger alarm to scaling engine", lager.Data{"trigger": trigger, "appMetric": appMetric})
				return false, appMetric
			}
		case ">=":
			if value < threshold {
				e.logger.Debug("should not send trigger alarm to scaling engine", lager.Data{"trigger": trigger, "appMetric": appMetric})
				return false, appMetric
			}
		case "<":
			if value >= threshold {
				e.logger.Debug("should not send trigger alarm to scaling engine", lager.Data{"trigger": trigger, "appMetric": appMetric})
				return false, appMetric
			}
		case "<=":
			if value > threshold {
				e.logger.Debug("should not send trigger alarm to scaling engine", lager.Data{"trigger": trigger, "appMetric": appMetric})
				return false, appMetric
			}
		}
	}
	return true, appMetric
}

func (e *Evaluator) retrieveAppMetrics(trigger *models.Trigger) ([]*models.AppMetric, error) {
	queryEndTime := time.Now()
	queryStartTime := queryEndTime.Add(0 - 2*trigger.BreachDuration())
	breachStartTime := queryEndTime.Add(0 - trigger.BreachDuration())

	appMetrics, err := e.queryAppMetrics(trigger.AppId, trigger.MetricType, queryStartTime.UnixNano(), queryEndTime.UnixNano(), db.ASC)
	if err != nil {
		e.logger.Error("retrieve-appMetrics", err, lager.Data{"trigger": trigger})
		return nil, err
	}

	e.logger.Debug("retrieve-appMetrics", lager.Data{"appMetrics": appMetrics})
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

	r := routes.NewRouter()
	scalingEngineRouter := r.CreateScalingEngineRoutes()

	path, err := scalingEngineRouter.Get(routes.ScaleRouteName).URLPath("appid", trigger.AppId)
	if err != nil {
		return fmt.Errorf("failed to create url ScaleRouteName, %s: %w", trigger.AppId, err)
	}

	resp, err := e.httpClient.Post(e.scalingEngineUrl+path.Path, "application/json", bytes.NewReader(jsonBytes))
	if err != nil {
		e.logger.Error("failed-send-trigger-alarm-request", err, lager.Data{"trigger": trigger})
		return err
	}

	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		e.logger.Error("failed-read-response-body-from-scaling-engine", err)
	}

	if resp.StatusCode == http.StatusOK {
		var scalingResult *models.AppScalingResult
		err = json.Unmarshal(respBody, &scalingResult)
		if err != nil {
			e.logger.Error("successfully-send-trigger-alarm, but received wrong response", err, lager.Data{"trigger": trigger, "responseBody": string(respBody)})
			return err
		}
		e.logger.Debug("successfully-send-trigger-alarm with trigger", lager.Data{"trigger": trigger, "responseBody": string(respBody)})
		if scalingResult.CooldownExpiredAt != 0 {
			e.setCoolDownExpired(trigger.AppId, scalingResult.CooldownExpiredAt)
		}
		return nil
	}
	err = fmt.Errorf("got %d when sending trigger alarm", resp.StatusCode)
	e.logger.Error("failed-send-trigger-alarm", err, lager.Data{"trigger": trigger, "responseBody": string(respBody)})
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
