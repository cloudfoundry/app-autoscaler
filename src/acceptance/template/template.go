package template

import (
	"bytes"
	"io/ioutil"
	"strconv"
)

// Parameter names to replace them to a value.
var (
	EndTimeValue   = "{endTimeValue}"
	MaxCount       = "{maxCount}"
	StartDateValue = "{startDateValue}"
	StartTimeValue = "{startTimeValue}"
	ReportInterval = "{reportInterval}"
)

var defaultValuesForPolicy = map[string]string{
	MaxCount:       "5",
	ReportInterval: "120",
	StartTimeValue: "00:00",
	StartDateValue: "2015-06-19",
	EndTimeValue:   "08:00",
}

func GeneratePolicy(template_file string, values map[string]string) ([]byte, error) {
	template, err := ioutil.ReadFile(template_file)
	if err != nil {
		return nil, err
	}

	merged_values := mergeValuesAndDefaultValues(values)

	for k, v := range merged_values {
		template = bytes.Replace(template, []byte(k), []byte(v), -1)
	}

	return template, nil
}

func mergeValuesAndDefaultValues(values map[string]string) map[string]string {
	var merge_values = map[string]string{}
	if values == nil {
		return defaultValuesForPolicy
	}

	for k, default_v := range defaultValuesForPolicy {
		value, ok := values[k]
		if ok {
			merge_values[k] = value
		} else {
			merge_values[k] = default_v
		}
	}

	return merge_values
}

func setValue(key string, value string, values map[string]string) map[string]string {
	if values == nil {
		values = map[string]string{}
	}
	values[key] = value
	return values
}
func SetStringValue(key string, value string, values map[string]string) map[string]string {
	new_values := setValue(key, value, values)
	return new_values
}

func SetIntValue(key string, value int, values map[string]string) map[string]string {
	new_values := setValue(key, strconv.Itoa(value), values)
	return new_values
}
