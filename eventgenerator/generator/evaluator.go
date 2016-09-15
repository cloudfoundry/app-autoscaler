package generator

import (
	"autoscaler/db"
	"autoscaler/eventgenerator/model"
	"bytes"
	"code.cloudfoundry.org/lager"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"
)

var validOperators []string = []string{">", ">=", "<", "<="}

type Evaluator struct {
	logger           lager.Logger
	httpClient       *http.Client
	scalingEngineUrl string
	triggerChan      chan []*model.Trigger
	doneChan         chan bool
	database         db.AppMetricDB
}

func NewEvaluator(logger lager.Logger, httpClient *http.Client, scalingEngineUrl string, triggerChan chan []*model.Trigger, database db.AppMetricDB) *Evaluator {
	return &Evaluator{
		logger:           logger.Session("Evaluator"),
		httpClient:       httpClient,
		scalingEngineUrl: scalingEngineUrl,
		triggerChan:      triggerChan,
		doneChan:         make(chan bool),
		database:         database,
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
func (e *Evaluator) doEvaluate(triggerArray []*model.Trigger) {

	for _, trigger := range triggerArray {
		threshold := trigger.Threshold
		operator := trigger.Operator

		if !e.isValidOperator(operator) {
			e.logger.Error("operator is invalid", nil, lager.Data{"trigger": trigger})
			continue
		}

		appMetricList, err := e.retrieveAppMetrics(trigger)
		if err != nil {
			return
		}
		if len(appMetricList) == 0 {
			e.logger.Debug("no available appmetric", lager.Data{"trigger": trigger})
			return
		}
		for _, appMetric := range appMetricList {
			if operator == ">" {
				if appMetric.Value <= threshold {
					e.logger.Debug("should not send trigger alarm to scaling engine", lager.Data{"trigger": trigger, "appMetric": appMetric})
					return
				}
			} else if operator == ">=" {
				if appMetric.Value < threshold {
					e.logger.Debug("should not send trigger alarm to scaling engine", lager.Data{"trigger": trigger, "appMetric": appMetric})
					return
				}
			} else if operator == "<" {
				if appMetric.Value >= threshold {
					e.logger.Debug("should not send trigger alarm to scaling engine", lager.Data{"trigger": trigger, "appMetric": appMetric})
					return
				}
			} else if operator == "<=" {
				if appMetric.Value > threshold {
					e.logger.Debug("should not send trigger alarm to scaling engine", lager.Data{"trigger": trigger, "appMetric": appMetric})
					return
				}
			}
		}

		e.logger.Info("send trigger alarm to scaling engine", lager.Data{"trigger": trigger})
		e.sendTriggerAlarm(trigger)

	}

}
func (e *Evaluator) retrieveAppMetrics(trigger *model.Trigger) ([]*model.AppMetric, error) {
	appId := trigger.AppId
	metricType := trigger.MetricType
	breachDuration := trigger.BreachDuration
	endTime := time.Now()
	startTime := endTime.Add(0 - breachDuration)
	appMetrics, err := e.database.RetrieveAppMetrics(appId, metricType, startTime.UnixNano(), endTime.UnixNano())
	if err != nil {
		e.logger.Error("retrieve appMetrics", err, lager.Data{"trigger": trigger})
		return nil, err
	}
	e.logger.Debug("appMetrics", lager.Data{"appMetrics": appMetrics})
	return appMetrics, nil
}
func (e *Evaluator) sendTriggerAlarm(trigger *model.Trigger) {
	url := e.scalingEngineUrl
	jsonBytes, jsonEncodeError := json.Marshal(trigger)
	if jsonEncodeError != nil {
		e.logger.Error("failed to json.Marshal trigger", jsonEncodeError)
	}
	path := "/v1/apps/" + trigger.AppId + "/scale"
	resp, respErr := e.httpClient.Post(url+path, "", bytes.NewReader(jsonBytes))
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
