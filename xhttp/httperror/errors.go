package httperror

import (
	"fmt"
	"net/http"

	"github.com/go-phorce/dolly/xhttp/header"
	"github.com/ugorji/go/codec"
)

// Error represents a single error from API.
type Error struct {
	// HTTPStatus contains the HTTP status code that should be used for this error
	HTTPStatus int `json:"-"`

	// Code identifies the particular error condition [for programatic consumers]
	Code string `json:"code"`

	// Message is an textual description of the error
	Message string `json:"message"`
}

// New builds a new Error instance, building the message string along the way
func New(status int, code string, msgFormat string, vals ...interface{}) error {
	return Error{
		HTTPStatus: status,
		Code:       code,
		Message:    fmt.Sprintf(msgFormat, vals...),
	}
}

// Error implements the standard error interface
func (e Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// ManyError identifies many errors from API.
type ManyError struct {
	// HTTPStatus contains the HTTP status code that should be used for this error
	HTTPStatus int `json:"-"`

	// Code identifies the particular error condition [for programatic consumers]
	Code string `json:"code,omitempty"`

	// Message is an textual description of the error
	Message string `json:"message,omitempty"`

	Errors map[string]Error `json:"errors,omitempty"`
}

func (m ManyError) Error() string {
	return fmt.Sprintf("%s: %s", m.Code, m.Message)
}

// NewMany builds new ManyError instance, build message string along the way
func NewMany(status int, code string, msgFormat string, vals ...interface{}) ManyError {
	return ManyError{
		HTTPStatus: status,
		Code:       code,
		Message:    fmt.Sprintf(msgFormat, vals...),
		Errors:     make(map[string]Error),
	}
}

// AddError add a single error to ManyError
func (m *ManyError) AddError(key string, err error) {
	if m.Errors == nil {
		m.Errors = make(map[string]Error)
	}
	if gErr, ok := err.(Error); ok {
		m.Errors[key] = gErr
	} else {
		m.Errors[key] = Error{Code: UnexpectedError, Message: err.Error()}
	}
}

// HasErrors check if ManyError has any nested error associated with it.
func (m ManyError) HasErrors() bool {
	return len(m.Errors) > 0
}

// WriteHTTPResponse implements how to serialize this error into a HTTP Response
func (e Error) WriteHTTPResponse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(header.ContentType, header.ApplicationJSON)
	w.WriteHeader(e.HTTPStatus)
	codec.NewEncoder(w, encoderHandle(shouldPrettyPrint(r))).Encode(e)
}

// WriteHTTPResponse implements how to serialize this error into a HTTP Response
func (m ManyError) WriteHTTPResponse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(header.ContentType, header.ApplicationJSON)
	w.WriteHeader(m.HTTPStatus)
	codec.NewEncoder(w, encoderHandle(shouldPrettyPrint(r))).Encode(m)
}
