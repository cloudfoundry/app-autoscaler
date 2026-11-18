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
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"

	"code.cloudfoundry.org/lager/v3"
	circuit "github.com/rubyist/circuitbreaker"
)

var validOperators = []string{">", ">=", "<", "<="}

// Reason is an optional wire field. Unknown/legacy ignore reasons fail closed;
// only an explicit dimension-specific refusal permits priority fallback.
type scalingEngineResult struct {
	models.AppScalingResult
	Reason string `json:"reason,omitempty"`
}

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
	maxMetricGap              time.Duration
}

func NewEvaluator(logger lager.Logger, httpClient *http.Client, scalingEngineUrl string, triggerChan chan []*models.Trigger,
	defaultBreachDurationSecs int, queryAppMetrics aggregator.QueryAppMetricsFunc, getBreaker func(string) *circuit.Breaker, setCoolDownExpired func(string, int64), collectionIntervals ...time.Duration) *Evaluator {
	collectionInterval := config.DefaultAggregatorExecuteInterval
	if len(collectionIntervals) > 0 && collectionIntervals[0] > 0 {
		collectionInterval = collectionIntervals[0]
	}
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
		maxMetricGap:              2 * collectionInterval,
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
	attempted := make(map[models.ScalingDimension]bool)
	for _, trigger := range triggerArray {
		dimension := trigger.ScalingDimension
		if dimension == "" {
			dimension = models.ScalingDimensionHorizontal
		}
		if attempted[dimension] {
			continue
		}
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
			attempted[dimension] = true
			trigger.MetricUnit = appMetricList[0].Unit
			e.logger.Info("send trigger alarm to scaling engine", lager.Data{"trigger": trigger, "last_metric": appMetric})

			var scalingResult *scalingEngineResult
			send := func() error {
				var sendErr error
				scalingResult, sendErr = e.sendTriggerAlarmResult(trigger)
				return sendErr
			}
			if appBreaker := e.getBreaker(trigger.AppId); appBreaker != nil {
				if appBreaker.Tripped() {
					e.logger.Info("circuit-tripped", lager.Data{"appId": trigger.AppId, "consecutiveFailures": appBreaker.ConsecFailures()})
				}
				err = appBreaker.Call(send, 0)
				if err != nil {
					e.logger.Error("circuit-alarm-failed", err, lager.Data{"appId": trigger.AppId})
				}
			} else {
				err = send()
				if err != nil {
					e.logger.Error("circuit-alarm-failed", err, lager.Data{"appId": trigger.AppId})
				}
			}
			// A rejected first dimension must not starve the other one. The
			// engine decides priority/bounds from current state. Stop once an
			// operation was accepted, its outcome is unknown, or the engine
			// reports an authoritative app-wide cooldown.
			if err != nil || !canEvaluateOtherDimension(scalingResult) {
				return
			}
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
	if trigger.MetricType == models.MetricNameEntitlementUtil {
		return entitlementBreachMetrics(appMetrics, breachStartTime, queryEndTime, e.maxMetricGap), nil
	}
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
	_, err := e.sendTriggerAlarmResult(trigger)
	return err
}

func (e *Evaluator) sendTriggerAlarmResult(trigger *models.Trigger) (*scalingEngineResult, error) {
	jsonBytes, err := json.Marshal(trigger)
	if err != nil {
		e.logger.Error("failed-marshal-trigger", err)
		return nil, err
	}

	r := routes.NewRouter()
	scalingEngineRouter := r.CreateScalingEngineRoutes()

	path, err := scalingEngineRouter.Get(routes.ScaleRouteName).URLPath("appid", trigger.AppId)
	if err != nil {
		return nil, fmt.Errorf("failed to create url ScaleRouteName, %s: %w", trigger.AppId, err)
	}

	resp, err := e.httpClient.Post(e.scalingEngineUrl+path.Path, "application/json", bytes.NewReader(jsonBytes))
	if err != nil {
		e.logger.Error("failed-send-trigger-alarm-request", err, lager.Data{"trigger": trigger})
		return nil, err
	}

	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		e.logger.Error("failed-read-response-body-from-scaling-engine", err)
		return nil, err
	}

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusAccepted {
		var scalingResult *scalingEngineResult
		err = json.Unmarshal(respBody, &scalingResult)
		if err != nil || scalingResult == nil {
			if err == nil {
				err = fmt.Errorf("scaling engine returned a null result")
			}
			e.logger.Error("successfully-send-trigger-alarm, but received wrong response", err, lager.Data{"trigger": trigger, "responseBody": string(respBody)})
			return nil, err
		}
		e.logger.Debug("successfully-send-trigger-alarm with trigger", lager.Data{"trigger": trigger, "responseBody": string(respBody)})
		if scalingResult.CooldownExpiredAt != 0 &&
			(scalingResult.Status == models.ScalingStatusSucceeded || scalingResult.Status == models.ScalingStatusIgnored) {
			e.setCoolDownExpired(trigger.AppId, scalingResult.CooldownExpiredAt)
		}
		return scalingResult, nil
	}
	err = fmt.Errorf("got %d when sending trigger alarm", resp.StatusCode)
	e.logger.Error("failed-send-trigger-alarm", err, lager.Data{"trigger": trigger, "responseBody": string(respBody)})
	return nil, err
}

func (e *Evaluator) isValidOperator(operator string) bool {
	for _, o := range validOperators {
		if o == operator {
			return true
		}
	}
	return false
}
