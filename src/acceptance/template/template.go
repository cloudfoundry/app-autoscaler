package template

import (
	"bytes"
	"io/ioutil"
	"strconv"
)

type TemplateValues map[string]string

// Parameter names to replace them to a value.
const (
	EndTimeValue   = "{endTimeValue}"
	MaxCount       = "{maxCount}"
	StartDateValue = "{startDateValue}"
	StartTimeValue = "{startTimeValue}"
	ReportInterval = "{reportInterval}"
)

var defaultValuesForPolicy = TemplateValues{
	MaxCount:       "5",
	ReportInterval: "120",
	StartTimeValue: "00:00",
	StartDateValue: "2015-06-19",
	EndTimeValue:   "08:00",
}

func GeneratePolicy(template_file string, values TemplateValues) ([]byte, error) {
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

func mergeValuesAndDefaultValues(values TemplateValues) TemplateValues {
	var merge_values = TemplateValues{}
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

func (values *TemplateValues) SetString(key string, value string) {
	(*values)[key] = value
}

func (values *TemplateValues) SetInt(key string, value int) {
	(*values)[key] = strconv.Itoa(value)
}
