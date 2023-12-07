package helpers

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"code.cloudfoundry.org/lager/v3"
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
	var unSupportedErr *json.UnsupportedTypeError
	var marshalErr *json.MarshalerError
	if err != nil {
		if errors.As(err, &unSupportedErr) || errors.As(err, &marshalErr) {
			tlf.Data = map[string]interface{}{"lager serialisation error": err.Error(), "data_dump": fmt.Sprintf("%#v", tlf.Data)}
			content, err = json.Marshal(tlf)
		}
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "%s", err.Error())
			content = []byte("{}")
		}
	}
	return content
}
