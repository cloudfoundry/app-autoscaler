package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const cfResourceNotFound = 10010

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type CFErrorResponse struct {
	Description string `json:"description"`
	ErrorCode   string `json:"error_code"`
	Code        int    `json:"code"`
}

var CfResourceNotFound = &CfError{Errors: []CfErrorItem{{Detail: "App usage event not found", Title: "CF-ResourceNotFound", Code: cfResourceNotFound}}}
var CfInternalServerError = &CfError{Errors: []CfErrorItem{{Detail: "An unexpected, uncaught error occurred; the CC logs will contain more information", Title: "UnknownError", Code: 10001}}}
var _ error = &CfError{}
var ErrInvalidJson = fmt.Errorf("invalid error json")

func NewCfError(url string, resourceId string, statusCode int, respBody []byte) error {
	var cfError = &CfError{}
	err := json.Unmarshal(respBody, &cfError)
	if err != nil {
		return fmt.Errorf("failed to unmarshal id:%s error status '%d' body:'%s' : %w", resourceId, statusCode, truncateString(string(respBody), 512), err)
	}
	cfError.ResourceId = resourceId
	cfError.StatusCode = statusCode
	cfError.url = url

	if !cfError.IsValid() {
		return fmt.Errorf("invalid cfError: resource %s status:%d body:%s :%w", resourceId, statusCode, truncateString(string(respBody), 512), ErrInvalidJson)
	}
	return cfError
}

// CfError cf V3 Error type
type CfError struct {
	Errors     []CfErrorItem `json:"errors"`
	StatusCode int
	ResourceId string
	url        string
}

type CfErrorItem struct {
	Code   int    `json:"code"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

func (c *CfError) Error() string {
	var errs []string
	message := "None found"
	for _, errorItem := range c.Errors {
		errorsString := fmt.Sprintf("['%s' code: %d, Detail: '%s']", errorItem.Title, errorItem.Code, errorItem.Detail)
		errs = append(errs, errorsString)
	}
	if len(errs) > 0 {
		message = strings.Join(errs, ", ")
	}
	return fmt.Sprintf("cf api Error url='%s', resourceId='%s': %s", c.url, c.ResourceId, message)
}

func (c *CfError) IsNotFound() bool {
	if c.IsValid() {
		for _, item := range c.Errors {
			if item.Code == cfResourceNotFound {
				return true
			}
		}
	}
	return false
}

func (c *CfError) IsValid() bool {
	return c != nil && len(c.Errors) > 0
}

func truncateString(stringToTrunk string, length int) string {
	if len(stringToTrunk) > length {
		return stringToTrunk[:length]
	}
	return stringToTrunk
}

func IsNotFound(err error) bool {
	var cfError *CfError
	return errors.As(err, &cfError) && cfError.IsNotFound()
}
