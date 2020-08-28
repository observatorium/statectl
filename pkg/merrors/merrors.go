package merrors

import (
	"bytes"
	"fmt"
)

// Error type implements the error interface, and contains the
// Errors used to construct it.
type Error []error

func New() *Error { return &Error{} }

// Error returns a concatenated string of the contained errors
func (es Error) Error() string {
	var buf bytes.Buffer

	if len(es) > 1 {
		fmt.Fprintf(&buf, "%d errors: ", len(es))
	}

	for i, err := range es {
		if i != 0 {
			buf.WriteString("; ")
		}
		buf.WriteString(err.Error())
	}

	return buf.String()
}

// Add adds the error to the error list if it is not nil.
func (es *Error) Add(err error) {
	if err == nil {
		return
	}
	if merr, ok := err.(Error); ok {
		*es = append(*es, merr...)
	} else {
		*es = append(*es, err)
	}
}

// Err returns the error list as an error or nil if it is empty.
func (es Error) Err() error {
	if len(es) == 0 {
		return nil
	}
	return es
}
