package header

const (
	// Accept is HTTP header for "Accept"
	Accept = "Accept"
	// ApplicationJSON is HTTP header value for "application/json"
	ApplicationJSON = "application/json"
	// ApplicationJoseJSON is HTTP header value for "application/jose+json"
	ApplicationJoseJSON = "application/jose+json"
	// ApplicationTimestampQuery is HTTP header value for RFC3161 Timestamp request
	ApplicationTimestampQuery = "application/timestamp-query"
	// ApplicationTimestampReply is HTTP header value for RFC3161 Timestamp response
	ApplicationTimestampReply = "application/timestamp-reply"
	// ContentDisposition is HTTP header for "Content-Disposition"
	ContentDisposition = "Content-Disposition"
	// ContentLength is HTTP header for "Content-Length"
	ContentLength = "Content-Length"
	// ContentType is HTTP header for "Content-Type"
	ContentType = "Content-Type"
	// Link is HTTP header for "Link"
	Link = "Link"
	// Location is HTTP header for "Location"
	Location = "Location"
	// ReplayNonce is HTTP header for "Replay-Nonce"
	ReplayNonce = "Replay-Nonce"
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
	// XForwardedProto contains the protocol
	XForwardedProto = "X-Forwarded-Proto"
)
