package models

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type AppNotFoundErr struct {
	message string
}

func NewAppNotFoundErr(message string) *AppNotFoundErr {
	return &AppNotFoundErr{
		message: message,
	}
}
func (e *AppNotFoundErr) Error() string {
	return e.message
}

type CFErrorResponse struct {
	Description string `json:"description"`
	ErrorCode   string `json:"error_code"`
	Code        int    `json:"code"`
}
