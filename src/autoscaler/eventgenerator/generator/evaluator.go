package generator

import (
	"autoscaler/db"
	"autoscaler/models"
	"autoscaler/routes"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"code.cloudfoundry.org/lager"
)

var validOperators []string = []string{">", ">=", "<", "<="}

type Evaluator struct {
	logger                    lager.Logger
	httpClient                *http.Client
	scalingEngineUrl          string
	triggerChan               chan []*models.Trigger
	doneChan                  chan bool
	database                  db.AppMetricDB
	defaultBreachDurationSecs int
}

func NewEvaluator(logger lager.Logger, httpClient *http.Client, scalingEngineUrl string, triggerChan chan []*models.Trigger,
	database db.AppMetricDB, defaultBreachDurationSecs int) *Evaluator {
	return &Evaluator{
		logger:                    logger.Session("Evaluator"),
		httpClient:                httpClient,
		scalingEngineUrl:          scalingEngineUrl,
		triggerChan:               triggerChan,
		doneChan:                  make(chan bool),
		database:                  database,
		defaultBreachDurationSecs: defaultBreachDurationSecs,
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
			triggerToSent := *trigger
			triggerToSent.MetricUnit = appMetricList[0].Unit
			e.logger.Info("send trigger alarm to scaling engine", lager.Data{"trigger": trigger})
			e.sendTriggerAlarm(&triggerToSent)
			return
		}
	}

}

func (e *Evaluator) retrieveAppMetrics(trigger *models.Trigger) ([]*models.AppMetric, error) {
	endTime := time.Now()
	startTime := endTime.Add(0 - trigger.BreachDuration(e.defaultBreachDurationSecs))
	appMetrics, err := e.database.RetrieveAppMetrics(trigger.AppId, trigger.MetricType, startTime.UnixNano(), endTime.UnixNano())
	if err != nil {
		e.logger.Error("retrieve appMetrics", err, lager.Data{"trigger": trigger})
		return nil, err
	}
	e.logger.Debug("appMetrics", lager.Data{"appMetrics": appMetrics})
	return appMetrics, nil
}

func (e *Evaluator) sendTriggerAlarm(trigger *models.Trigger) {
	jsonBytes, jsonEncodeError := json.Marshal(trigger)
	if jsonEncodeError != nil {
		e.logger.Error("failed to json.Marshal trigger", jsonEncodeError)
	}
	path, _ := routes.ScalingEngineRoutes().Get(routes.ScaleRouteName).URLPath("appid", trigger.AppId)
	resp, respErr := e.httpClient.Post(e.scalingEngineUrl+path.Path, "application/json", bytes.NewReader(jsonBytes))
	if respErr != nil {
		e.logger.Error("http reqeust error,failed to send trigger alarm", respErr, lager.Data{"trigger": trigger})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		e.logger.Info("successfully send trigger alarm", lager.Data{"trigger": trigger})
	} else {
		respBody, readError := ioutil.ReadAll(resp.Body)
		if readError != nil {
			e.logger.Error("failed to read body from scaling engine's response", readError)
		}
		e.logger.Error("scaling engine error,failed to send trigger alarm", nil, lager.Data{"responseCode": resp.StatusCode, "responseBody": respBody})
	}
}
func (e *Evaluator) isValidOperator(operator string) bool {
	for _, o := range validOperators {
		if o == operator {
			return true
		}
	}
	return false
}
