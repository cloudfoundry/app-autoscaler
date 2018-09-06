package helpers

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
