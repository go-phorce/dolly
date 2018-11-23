package httperror_test

import (
	"net/http"
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

func Test_StatusCodes(t *testing.T) {
	tcases := []struct {
		httpErr   *httperror.Error
		expStatus int
		expMsg    string
	}{
		{httperror.WithInvalidParam("1"), http.StatusBadRequest, "invalid_parameter: 1"},
		{httperror.WithInvalidJSON("1"), http.StatusBadRequest, "invalid_json: 1"},
		{httperror.WithBadNonce("1"), http.StatusBadRequest, "bad_nonce: 1"},
		{httperror.WithInvalidRequest("1"), http.StatusBadRequest, "invalid_request: 1"},
		{httperror.WithMalformed("1"), http.StatusBadRequest, "malformed: 1"},
		{httperror.WithInvalidContentType("1"), http.StatusBadRequest, "invalid_content_type: 1"},
		{httperror.WithContentLengthRequired(), http.StatusBadRequest, "content_length_required: Content-Length header not provided"},
		{httperror.WithNotFound("1"), http.StatusNotFound, "not_found: 1"},
		{httperror.WithRequestTooLarge("1"), http.StatusBadRequest, "request_too_large: 1"},
		{httperror.WithFailedToReadRequestBody("1"), http.StatusInternalServerError, "request_body: 1"},
		{httperror.WithRateLimitExceeded("1"), http.StatusTooManyRequests, "rate_limit_exceeded: 1"},
		{httperror.WithUnexpected("1"), http.StatusInternalServerError, "unexpected: 1"},
		{httperror.WithForbidden("1"), http.StatusForbidden, "forbidden: 1"},
		{httperror.WithUnauthorized("1"), http.StatusUnauthorized, "unauthorized: 1"},
		{httperror.WithAccountNotFound("1"), http.StatusForbidden, "account_not_found: 1"},
		{httperror.WithNotReady("1"), http.StatusForbidden, "not_ready: 1"},
	}
	for _, tc := range tcases {
		t.Run(tc.httpErr.Code, func(t *testing.T) {
			assert.Equal(t, tc.expStatus, tc.httpErr.HTTPStatus)
			assert.Equal(t, tc.expMsg, tc.httpErr.Error())
		})
	}
}
