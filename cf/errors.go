package cf

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/cloudfoundry/go-cfclient/v3/resource"
)

const (
	cfResourceNotFound = 10010
	cfNotAuthenticated = 10002
	cfNotAuthorised    = 10003
)

var (
	ErrUnauthorized       = errors.New("Unauthorized")
	ErrInvalidTokenFormat = errors.New("invalid token format")
	ErrInvalidJson        = errors.New("invalid error json")
)

type CFErrorResponse struct {
	Description string `json:"description"`
	ErrorCode   string `json:"error_code"`
	Code        int    `json:"code"`
}

var (
	CfResourceNotFound    = &CfError{Errors: []CfErrorItem{{Detail: "App usage event not found", Title: "CF-ResourceNotFound", Code: cfResourceNotFound}}}
	CfInternalServerError = &CfError{Errors: []CfErrorItem{{Detail: "An unexpected, uncaught error occurred; the CC logs will contain more information", Title: "UnknownError", Code: 10001}}}
	CfNotAuthenticated    = &CfError{Errors: []CfErrorItem{{Detail: "No auth token was given, but authentication is required for this endpoint", Title: "CF-NotAuthenticated", Code: cfNotAuthenticated}}}
	CfNotAuthorized       = &CfError{Errors: []CfErrorItem{{Detail: "The authenticated user does not have permission to perform this operation", Title: "CF-NotAuthorized", Code: cfNotAuthorised}}}

	_ error = &CfError{}
)

func NewCfError(url string, resourceId string, statusCode int, respBody []byte) error {
	var cfError = &CfError{}
	err := json.Unmarshal(respBody, &cfError)
	if err != nil {
		return fmt.Errorf("failed to unmarshal id:%s error status '%d' body:'%s' : %w", resourceId, statusCode, truncateString(string(respBody), 512), err)
	}
	cfError.ResourceId = resourceId
	cfError.StatusCode = statusCode
	cfError.Url = url

	if !cfError.IsValid() {
		return fmt.Errorf("invalid cfError: resource %s status:%d body:%s :%w", resourceId, statusCode, truncateString(string(respBody), 512), ErrInvalidJson)
	}
	return cfError
}

// CfError cf V3 Error type
type CfError struct {
	Errors     ErrorItems `json:"errors"`
	StatusCode int
	ResourceId string
	Url        string
}
type ErrorItems []CfErrorItem

func (e ErrorItems) hasError(errorCode int) bool {
	for _, item := range e {
		if item.Code == errorCode {
			return true
		}
	}
	return false
}

type CfErrorItem struct {
	Code   int    `json:"code"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

func (c *CfError) Error() string {
	if len(c.Errors) == 0 {
		return fmt.Sprintf("cf api Error url='%s', resourceId='%s': None found", c.Url, c.ResourceId)
	}
	errs := make([]string, len(c.Errors))
	for i, item := range c.Errors {
		errs[i] = fmt.Sprintf("['%s' code: %d, Detail: '%s']", item.Title, item.Code, item.Detail)
	}
	return fmt.Sprintf("cf api Error url='%s', resourceId='%s': %s", c.Url, c.ResourceId, strings.Join(errs, ", "))
}

func (c *CfError) IsValid() bool {
	return c != nil && len(c.Errors) > 0
}

func (c *CfError) ContainsError(errorCode int) bool {
	if c.IsValid() {
		return c.Errors.hasError(errorCode)
	}
	return false
}

func (c *CfError) IsNotFound() bool {
	return c.ContainsError(cfResourceNotFound)
}
func (c *CfError) IsNotAuthorised() bool {
	return c.ContainsError(cfNotAuthorised)
}

func (c *CfError) IsNotAuthenticated() bool {
	return c.ContainsError(cfNotAuthenticated)
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

func MapCFClientError(err error) error {
	if err == nil {
		return nil
	}

	var cfHTTPErr resource.CloudFoundryHTTPError
	if errors.As(err, &cfHTTPErr) {
		return &CfError{
			StatusCode: cfHTTPErr.StatusCode,
			Errors: []CfErrorItem{{
				Title:  cfHTTPErr.Status,
				Detail: string(cfHTTPErr.Body),
			}},
		}
	}

	var cfClientErr resource.CloudFoundryError
	if errors.As(err, &cfClientErr) {
		return &CfError{
			StatusCode: httpStatusFromCFCode(cfClientErr.Code),
			Errors: []CfErrorItem{{
				Code:   cfClientErr.Code,
				Title:  cfClientErr.Title,
				Detail: cfClientErr.Detail,
			}},
		}
	}

	var cfClientErrs resource.CloudFoundryErrors
	if errors.As(err, &cfClientErrs) {
		items := make([]CfErrorItem, len(cfClientErrs.Errors))
		statusCode := 0
		for i, e := range cfClientErrs.Errors {
			items[i] = CfErrorItem{
				Code:   e.Code,
				Title:  e.Title,
				Detail: e.Detail,
			}
			if statusCode == 0 {
				statusCode = httpStatusFromCFCode(e.Code)
			}
		}
		return &CfError{StatusCode: statusCode, Errors: items}
	}

	return err
}

func httpStatusFromCFCode(cfCode int) int {
	switch cfCode {
	case cfResourceNotFound:
		return http.StatusNotFound
	case cfNotAuthenticated:
		return http.StatusUnauthorized
	case cfNotAuthorised:
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}
