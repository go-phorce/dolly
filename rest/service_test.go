package rest_test

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/go-phorce/dolly/rest"
	"github.com/go-phorce/dolly/testify/auditor"
	"github.com/go-phorce/dolly/xhttp/context"
	"github.com/go-phorce/dolly/xhttp/header"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/dig"
)

const testURL = "/v1/test"

// service defines the Status service
type service struct {
	server rest.Server
}

// NewService returns ane instances of the Status service
func NewService(
	server rest.Server,
) rest.Service {
	if server == nil {
		panic("invalid parameter to status.NewService")
	}

	return &service{
		server: server,
	}
}

// Name returns the service name
func (s *service) Name() string {
	return "testService"
}

// IsReady indicates that the service is ready to serve its end-points
func (s *service) IsReady() bool {
	return true
}

// Close the subservices and it's resources
func (s *service) Close() {
}

// Register adds the endpoints to the overall URL router
func (s *service) Register(r rest.Router) {
	r.GET(testURL, testHandler(s))
}

func testHandler(s *service) rest.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ rest.Params) {
		context.ForRequest(r)

		w.Header().Set(header.ContentType, header.TextPlain)
		fmt.Fprintf(w, "URL: %s\n", r.URL)
		fmt.Fprintf(w, "Method: %s\n", r.Method)
		fmt.Fprintf(w, "Agent: %s\n", r.UserAgent())
		fmt.Fprintf(w, "RemoteAddr\n: %s", r.RemoteAddr)
	}
}

type ctx struct {
}

func (c *ctx) SetHeaders(r *http.Request) {
}

func Test_ServerWithServicesOverHTTP(t *testing.T) {
	cfg := &serverConfig{
		BindAddr: ":8088",
	}

	ioc := dig.New()
	ioc.Provide(func() rest.HTTPServerConfig {
		return cfg
	})

	server, err := rest.New("test", "v1.0.123", ioc)
	require.NoError(t, err)
	require.NotNil(t, server)

	svc := NewService(server)
	server.AddService(svc)

	//	assert.NotNil(t, server.AddService())
	err = server.StartHTTP()
	require.NoError(t, err)

	for i := 0; i < 10 && !server.IsReady(); i++ {
		time.Sleep(100 * time.Millisecond)
	}
	require.True(t, server.IsReady())

	testService(t, server)

	server.StopHTTP()
}

func Test_ServerWithServicesOverHTTPS(t *testing.T) {
	tlsCfg, err := newTLSConfig(true)
	require.NoError(t, err)

	tlsInfo, tlsloader, err := createTLSInfo(tlsCfg)
	require.NoError(t, err)

	defer tlsloader.Close()

	cfg := &serverConfig{
		BindAddr: ":8443",
	}

	ioc := dig.New()
	ioc.Provide(func() rest.HTTPServerConfig {
		return cfg
	})
	ioc.Provide(func() rest.Auditor {
		return auditor.NewInMemory()
	})
	ioc.Provide(func() *tls.Config {
		return tlsInfo
	})
	server, err := rest.New("test", "v1.0.123", ioc)
	require.NoError(t, err)
	require.NotNil(t, server)
	assert.Equal(t, "https", server.Protocol())

	svc := NewService(server)
	server.AddService(svc)

	//	assert.NotNil(t, server.AddService())
	err = server.StartHTTP()
	require.NoError(t, err)

	defer server.StopHTTP()

	for i := 0; i < 10 && !server.IsReady(); i++ {
		time.Sleep(100 * time.Millisecond)
	}
	require.True(t, server.IsReady())

	_, err = http.Get(fmt.Sprintf("%s://localhost:%s/v1/test", server.Protocol(), server.Port()))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "certificate signed by unknown authority")

	// testService(t, server)
}

func testService(t *testing.T, server rest.Server) {
	resp, err := http.Get(fmt.Sprintf("%s://localhost:%s/v1/test", server.Protocol(), server.Port()))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	b, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	txt := string(b)
	assert.Contains(t, txt, "Method: GET")
}
