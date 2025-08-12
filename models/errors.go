package models

import (
	"errors"
	"fmt"
)

var ErrUnimplemented error = errors.New("🚧 To-do: This is still uninmplemented")

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type InvalidArgumentError struct {
	Param string
	Value any
	Msg   string
}

func (e *InvalidArgumentError) Error() string {
	msg := fmt.Sprintf(
		"invalid argument '%s': %v - %s\n\t%s",
		e.Param, e.Value, e.Msg,
		"⚠️ This is probably a bug in the code, please report it to the developers.")
	return msg
}
