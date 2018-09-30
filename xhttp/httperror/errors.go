package httperror

import (
	"fmt"
	"net/http"
	"strings"

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

	// Cause is the original error
	Cause error `json:"-"`
}

// New builds a new Error instance, building the message string along the way
func New(status int, code string, msgFormat string, vals ...interface{}) *Error {
	return &Error{
		HTTPStatus: status,
		Code:       code,
		Message:    fmt.Sprintf(msgFormat, vals...),
	}
}

// WithInvalidParam for builds a new Error instance with InvalidParam code
func WithInvalidParam(msgFormat string, vals ...interface{}) *Error {
	return New(http.StatusBadRequest, InvalidParam, msgFormat, vals...)
}

// WithInvalidJSON for builds a new Error instance with InvalidJSON code
func WithInvalidJSON(msgFormat string, vals ...interface{}) *Error {
	return New(http.StatusBadRequest, InvalidJSON, msgFormat, vals...)
}

// WithInvalidRequest for builds a new Error instance with InvalidRequest code
func WithInvalidRequest(msgFormat string, vals ...interface{}) *Error {
	return New(http.StatusBadRequest, InvalidRequest, msgFormat, vals...)
}

// WithNotFound for builds a new Error instance with NotFound code
func WithNotFound(msgFormat string, vals ...interface{}) *Error {
	return New(http.StatusNotFound, NotFound, msgFormat, vals...)
}

// WithRequestTooLarge for builds a new Error instance with RequestTooLarge code
func WithRequestTooLarge(msgFormat string, vals ...interface{}) *Error {
	return New(http.StatusBadRequest, RequestTooLarge, msgFormat, vals...)
}

// WithFailedToReadRequestBody for builds a new Error instance with FailedToReadRequestBody code
func WithFailedToReadRequestBody(msgFormat string, vals ...interface{}) *Error {
	return New(http.StatusInternalServerError, FailedToReadRequestBody, msgFormat, vals...)
}

// WithRateLimitExceeded for builds a new Error instance with RateLimitExceeded code
func WithRateLimitExceeded(msgFormat string, vals ...interface{}) *Error {
	return New(http.StatusTooManyRequests, RateLimitExceeded, msgFormat, vals...)
}

// WithUnexpected for builds a new Error instance with Unexpected code
func WithUnexpected(msgFormat string, vals ...interface{}) *Error {
	return New(http.StatusInternalServerError, Unexpected, msgFormat, vals...)
}

// WithForbidden for builds a new Error instance with Forbidden code
func WithForbidden(msgFormat string, vals ...interface{}) *Error {
	return New(http.StatusForbidden, Forbidden, msgFormat, vals...)
}

// WithNotReady for builds a new Error instance with NotReady code
func WithNotReady(msgFormat string, vals ...interface{}) *Error {
	return New(http.StatusForbidden, NotReady, msgFormat, vals...)
}

// WithCause adds the cause error
func (e *Error) WithCause(err error) *Error {
	e.Cause = err
	return e
}

// Error implements the standard error interface
func (e *Error) Error() string {
	if e == nil {
		return "nil"
	}
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

	Errors map[string]*Error `json:"errors,omitempty"`
}

func (m *ManyError) Error() string {
	if m == nil {
		return "nil"
	}
	if m.Code != "" {
		return fmt.Sprintf("%s: %s", m.Code, m.Message)
	}

	var errs []string
	for _, e := range m.Errors {
		errs = append(errs, e.Error())
	}

	return strings.Join(errs, ";")
}

// NewMany builds new ManyError instance, build message string along the way
func NewMany(status int, code string, msgFormat string, vals ...interface{}) *ManyError {
	return &ManyError{
		HTTPStatus: status,
		Code:       code,
		Message:    fmt.Sprintf(msgFormat, vals...),
		Errors:     make(map[string]*Error),
	}
}

// Add a single error to ManyError
func (m *ManyError) Add(key string, err error) *ManyError {
	if m == nil {
		m = new(ManyError)
	}
	if m.Errors == nil {
		m.Errors = make(map[string]*Error)
	}
	if gErr, ok := err.(*Error); ok {
		m.Errors[key] = gErr
	} else {
		m.Errors[key] = &Error{Code: Unexpected, Message: err.Error(), Cause: err}
	}
	return m
}

// HasErrors check if ManyError has any nested error associated with it.
func (m *ManyError) HasErrors() bool {
	return len(m.Errors) > 0
}

// WriteHTTPResponse implements how to serialize this error into a HTTP Response
func (e *Error) WriteHTTPResponse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(header.ContentType, header.ApplicationJSON)
	w.WriteHeader(e.HTTPStatus)
	codec.NewEncoder(w, encoderHandle(shouldPrettyPrint(r))).Encode(e)
}

// WriteHTTPResponse implements how to serialize this error into a HTTP Response
func (m *ManyError) WriteHTTPResponse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(header.ContentType, header.ApplicationJSON)
	w.WriteHeader(m.HTTPStatus)
	codec.NewEncoder(w, encoderHandle(shouldPrettyPrint(r))).Encode(m)
}
