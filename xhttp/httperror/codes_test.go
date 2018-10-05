package httperror_test

import (
	"testing"

	"github.com/go-phorce/dolly/xhttp/httperror"
	"github.com/stretchr/testify/assert"
)

func Test_ErrorCodes(t *testing.T) {
	assert.Equal(t, "account_not_found", httperror.AccountNotFound)
	assert.Equal(t, "bad_nonce", httperror.BadNonce)
	assert.Equal(t, "connection", httperror.Connection)
	assert.Equal(t, "content_length_required", httperror.ContentLengthRequired)
	assert.Equal(t, "forbidden", httperror.Forbidden)
	assert.Equal(t, "invalid_content_type", httperror.InvalidContentType)
	assert.Equal(t, "invalid_json", httperror.InvalidJSON)
	assert.Equal(t, "invalid_parameter", httperror.InvalidParam)
	assert.Equal(t, "invalid_request", httperror.InvalidRequest)
	assert.Equal(t, "malformed", httperror.Malformed)
	assert.Equal(t, "not_found", httperror.NotFound)
	assert.Equal(t, "not_ready", httperror.NotReady)
	assert.Equal(t, "rate_limit_exceeded", httperror.RateLimitExceeded)
	assert.Equal(t, "request_body", httperror.FailedToReadRequestBody)
	assert.Equal(t, "request_too_large", httperror.RequestTooLarge)
	assert.Equal(t, "unauthorized", httperror.Unauthorized)
	assert.Equal(t, "unexpected", httperror.Unexpected)
}
