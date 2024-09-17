package helpers

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"

	"code.cloudfoundry.org/lager/v3"
)

const postgresDbURLPattern = `^(postgres|postgresql):\/\/(.+):(.+)@([\da-zA-Z\.-]+)(:[\d]{4,5})?\/(.+)`

type JSONRedacterWithURLCred struct {
	jsonRedacter   *lager.JSONRedacter
	urlCredMatcher *regexp.Regexp
}

func NewJSONRedacterWithURLCred(keyPatterns []string, valuePatterns []string) (*JSONRedacterWithURLCred, error) {
	jsonRedacter, err := lager.NewJSONRedacter(keyPatterns, valuePatterns)
	if err != nil {
		return nil, err
	}
	urlCredMatcher, err := regexp.Compile(postgresDbURLPattern)
	if err != nil {
		return nil, err
	}
	return &JSONRedacterWithURLCred{
		jsonRedacter:   jsonRedacter,
		urlCredMatcher: urlCredMatcher,
	}, nil
}

func (r JSONRedacterWithURLCred) Redact(data []byte) []byte {
	var jsonBlob interface{}
	if len(data) == 0 {
		_, _ = fmt.Fprintf(os.Stderr, "%s", "unncessary to redact empty json object")
		return data
	}
	err := json.Unmarshal(data, &jsonBlob)
	if err != nil {
		return errorToBytes(err)
	}
	r.redactValue(&jsonBlob)

	data, err = json.Marshal(jsonBlob)
	if err != nil {
		return errorToBytes(err)
	}

	return r.jsonRedacter.Redact(data)
}

func (r JSONRedacterWithURLCred) redactValue(data *interface{}) interface{} {
	if data == nil {
		return data
	}

	if a, ok := (*data).([]interface{}); ok {
		r.redactArray(&a)
	} else if m, ok := (*data).(map[string]interface{}); ok {
		r.redactObject(&m)
	} else if s, ok := (*data).(string); ok {
		if r.urlCredMatcher.MatchString(s) {
			*data = r.urlCredMatcher.ReplaceAllString(s, `$1://$2:*REDACTED*@$4$5/$6`)
		}
	}
	return *data
}

func (r JSONRedacterWithURLCred) redactArray(data *[]interface{}) {
	for i := range *data {
		r.redactValue(&((*data)[i]))
	}
}

func (r JSONRedacterWithURLCred) redactObject(data *map[string]interface{}) {
	for k, v := range *data {
		val := v
		(*data)[k] = r.redactValue(&val)
	}
}

func errorToBytes(err error) []byte {
	var content []byte
	var errType *json.UnsupportedTypeError
	if errors.As(err, &errType) {
		data := map[string]interface{}{"lager serialisation error": errType.Error()}
		content, err = json.Marshal(data)
	}
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s", err.Error())
		return content
	}
	return content
}
