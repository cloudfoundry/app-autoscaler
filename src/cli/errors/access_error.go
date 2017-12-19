package errors

type AccessError struct {
	description string
}

func NewAccessError(description string) error {
	return &AccessError{
		description: description,
	}
}

func (err *AccessError) Error() string {
	return err.description
}
