package helpers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"code.cloudfoundry.org/lager"
)

type TimeLogFormat struct {
	lager.LogFormat
	LogTime string `json:"log_time"`
}

func NewTimeLogFormat(log lager.LogFormat) TimeLogFormat {
	floatTime, err := strconv.ParseFloat(log.Timestamp, 64)
	if err != nil {
		floatTime = 0.0
	}
	intTime := int64(floatTime)
	tm := time.Unix(intTime, 0)
	return TimeLogFormat{
		LogTime:   time.Time.Format(tm, time.RFC3339),
		LogFormat: log,
	}
}

func (tlf TimeLogFormat) ToJSON() []byte {
	content, err := json.Marshal(tlf)
	if err != nil {
		_, ok1 := err.(*json.UnsupportedTypeError)
		_, ok2 := err.(*json.MarshalerError)
		if ok1 || ok2 {
			tlf.Data = map[string]interface{}{"lager serialisation error": err.Error(), "data_dump": fmt.Sprintf("%#v", tlf.Data)}
			content, err = json.Marshal(tlf)
		}
		if err != nil {
			panic(err)
		}
	}
	return content
}
