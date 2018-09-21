package header

const (
	// Accept is HTTP header for "Accept"
	Accept = "Accept"
	// ContentType is HTTP header for "Content-Type"
	ContentType = "Content-Type"
	// ContentLength is HTTP header for "Content-Length"
	ContentLength = "Content-Length"
	// ContentDisposition is HTTP header for "Content-Disposition"
	ContentDisposition = "Content-Disposition"

	// ApplicationJSON is HTTP header value for "application/json"
	ApplicationJSON = "application/json"

	// ApplicationTimestampQuery is HTTP header value for RFC3161 Timestamp request
	ApplicationTimestampQuery = "application/timestamp-query"
	// ApplicationTimestampReply is HTTP header value for RFC3161 Timestamp response
	ApplicationTimestampReply = "application/timestamp-reply"

	// TextPlain is HTTP header value for "application/json"
	TextPlain = "text/plain"

	// XIdentity is HTTP header for "X-Identity" which is used for cross-role requests
	XIdentity = "X-Identity"
	// XHostname contains the name of the HTTP header to indicate which host requested the signature
	XHostname = "X-HostName"

	// XCorrelationID is HTTP header for "X-CorrelationID"
	XCorrelationID = "X-CorrelationID"
	// XFilename contains the name of the artifact to sign
	XFilename = "X-Filename"
)
