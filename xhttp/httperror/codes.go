package httperror

const (
	// InvalidParam is returned where a URL parameter, or other type of generalized parameters value is invalid.
	InvalidParam = "invalid_parameter"

	// InvalidJSON is returned when we were unable to parse a client supplied JSON Payload.
	InvalidJSON = "invalid_json"

	// InvalidRequest is returned when the request validation failed.
	InvalidRequest = "invalid_request"

	// NotFound is returned when the requested URL doesn't exist.
	NotFound = "not_found"

	// RequestTooLarge is returned when the client provided payload is larger than allowed for the particular resource.
	RequestTooLarge = "request_too_large"

	// FailedToReadRequestBody is returned when there's an error reading the HTTP body of the request.
	FailedToReadRequestBody = "request_body"

	// RateLimitExceeded is returned when the client has exceeded their request allotment.
	RateLimitExceeded = "rate_limit_exceeded"

	// UnexpectedError is returned when something went wrong.
	UnexpectedError = "unexpected"

	// Forbidden is returned when the client is not authorized to access the resource indicated.
	Forbidden = "forbidden"

	// NotReady is returned when the service is not ready to serve
	NotReady = "not_ready"
)
