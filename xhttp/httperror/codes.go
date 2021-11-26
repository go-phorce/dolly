package httperror

const (
	// AccountNotFound when requested account not found
	AccountNotFound = "account_not_found"
	// BadNonce is returned for bad nonce.
	BadNonce = "bad_nonce"
	// Conflict is returned whith 409 CONFLICT response code.
	Conflict = "conflict"
	// Connection is returned when connection failed.
	Connection = "connection"
	// ContentLengthRequired is returned when request does not specify ContentLength.
	ContentLengthRequired = "content_length_required"
	// FailedToReadRequestBody is returned when there's an error reading the HTTP body of the request.
	FailedToReadRequestBody = "request_body"
	// Forbidden is returned when the client is not authorized to access the resource indicated.
	Forbidden = "forbidden"
	// InvalidContentType is returned when request specifies invalid Content-Type.
	InvalidContentType = "invalid_content_type"
	// InvalidJSON is returned when we were unable to parse a client supplied JSON Payload.
	InvalidJSON = "invalid_json"
	// InvalidParam is returned where a URL parameter, or other type of generalized parameters value is invalid.
	InvalidParam = "invalid_parameter"
	// InvalidRequest is returned when the request validation failed.
	InvalidRequest = "invalid_request"
	// Malformed is returned when the request was malformed.
	Malformed = "malformed"
	// NotFound is returned when the requested URL doesn't exist.
	NotFound = "not_found"
	// NotReady is returned when the service is not ready to serve
	NotReady = "not_ready"
	// RateLimitExceeded is returned when the client has exceeded their request allotment.
	RateLimitExceeded = "rate_limit_exceeded"
	// RequestFailed is returned when an outbound request failed.
	RequestFailed = "request_failed"
	// RequestTooLarge is returned when the client provided payload is larger than allowed for the particular resource.
	RequestTooLarge = "request_too_large"
	// TooEarly is returned when the client makes requests too early.
	TooEarly = "too_early"
	// Unauthorized is for unauthorized access.
	Unauthorized = "unauthorized"
	// Unexpected is returned when something went wrong.
	Unexpected = "unexpected"
)
