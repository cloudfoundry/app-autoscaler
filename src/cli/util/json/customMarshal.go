package json

import (
	"bytes"
	"encoding/json"
)

func MarshalWithoutHTMLEscape(v interface{}) ([]byte, error) {

	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "\t")
	err := enc.Encode(v)
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil

}
