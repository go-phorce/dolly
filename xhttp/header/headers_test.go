package header_test

import (
	"testing"

	"github.com/ekspand/pkg/xhttp/header"
	"github.com/stretchr/testify/assert"
)

func Test_Headers(t *testing.T) {
	assert.Equal(t, "Accept", header.Accept)
	assert.Equal(t, "Content-Type", header.ContentType)
	assert.Equal(t, "Content-Disposition", header.ContentDisposition)
	assert.Equal(t, "application/json", header.ApplicationJSON)
	assert.Equal(t, "application/timestamp-query", header.ApplicationTimestampQuery)
	assert.Equal(t, "application/timestamp-reply", header.ApplicationTimestampReply)
	assert.Equal(t, "text/plain", header.TextPlain)
	assert.Equal(t, "X-Identity", header.XIdentity)
	assert.Equal(t, "X-HostName", header.XHostname)
	assert.Equal(t, "X-CorrelationID", header.XCorrelationID)
	assert.Equal(t, "X-Filename", header.XFilename)
}
