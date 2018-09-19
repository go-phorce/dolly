package httperror

const (
	// InvalidParam is returned where a URL parameter, or other type of generalized parameters value is invalid.
	InvalidParam = "Invalid parameter"

	// InvalidJSON is returned when we were unable to parse a client supplied JSON Payload.
	InvalidJSON = "Invalid JSON"

	// InvalidRequest is returned when the request validation failed.
	InvalidRequest = "Invalid request"

	// NotFound is returned when the requested URL doesn't exist.
	NotFound = "Not found"

	// RequestTooLarge is returned when the client provided payload is larger than allowed for the particular resource.
	RequestTooLarge = "Request too large"

	// FailedToReadRequestBody is returned when there's an error reading the HTTP body of the request.
	FailedToReadRequestBody = "Failed to read request body"

	// RateLimitExceeded is returned when the client has exceeded their request allotment.
	RateLimitExceeded = "Rate limit exceeded"

	// UnexpectedError is returned when something went wrong.
	UnexpectedError = "Unexpected"

	// Forbidden is returned when the client is not authorized to access the resource indicated. [e.g. the client cert doesn't have a Salesforce.com Organization].
	Forbidden = "Forbidden"

	// NotReady is returned when the service is not ready to serve
	NotReady = "Not ready"
)
