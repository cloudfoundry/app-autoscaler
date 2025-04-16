package models

import "fmt"

var ErrUnimplemented error = fmt.Errorf("🚧 To-do: This is still uninmplemented")

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
