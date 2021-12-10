package pkg

// Error represents a handler error. It provides methods for a HTTP status
// code and embeds the built-in error interface.
type Error interface {
	error
	Status() int
}

// NotifErr represents an error with an associated HTTP status code.
type NotifErr struct {
	Code int
	Err  error
}

// Error allows NotifErr to satisfy the error interface.
func (e NotifErr) Error() string {
	return e.Err.Error()
}

// Status returns the HTTP status code.
func (e NotifErr) Status() int {
	return e.Code
}
