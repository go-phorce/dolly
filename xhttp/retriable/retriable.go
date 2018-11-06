package retriable

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/go-phorce/dolly/algorithms/slices"
	"github.com/go-phorce/dolly/xhttp/httperror"
	"github.com/go-phorce/dolly/xlog"
	"github.com/juju/errors"
	"golang.org/x/net/http2"
)

var logger = xlog.NewPackageLogger("github.com/go-phorce/dolly/xhttp", "retriable")

const (
	// Success returned when request succeeded
	Success = "success"
	// NotFound returned when request returned 404
	NotFound = "not-found"
	// LimitExceeded returned when retry limit exceeded
	LimitExceeded = "limit-exceeded"
	// Cancelled returned when request was cancelled or timed out
	Cancelled = "cancelled"
	// NonRetriableError returned when non-retriable error occured
	NonRetriableError = "non-retriable"
)

// ShouldRetry specifies a policy for handling retries. It is called
// following each request with the response, error values returned by
// the http.Client and the number of already made retries.
// If ShouldRetry returns false, the Client stops retrying
// and returns the response to the caller. The
// Client will close any response body when retrying, but if the retriable is
// aborted it is up to the caller to properly close any response body before returning.
type ShouldRetry func(r *http.Request, resp *http.Response, err error, retries int) (bool, time.Duration, string)

// Policy represents the retriable policy
type Policy struct {

	// Retries specifies a map of HTTP Status code to ShouldRetry function,
	// 0 status code indicates a connection related error (network, TLS, DNS etc.)
	Retries map[int]ShouldRetry

	// Maximum number of retries.
	TotalRetryLimit int

	RequestTimeout time.Duration
}

// A ClientOption modifies the default behavior of Client.
type ClientOption interface {
	applyOption(*Client)
}

type optionFunc func(*Client)

func (f optionFunc) applyOption(opts *Client) { f(opts) }

type clientOptions struct {
}

// WithName is a ClientOption that specifies client's name for logging purposes.
//
//   retriable.New(retriable.WithName("tlsclient"))
//
// This option cannot be provided for constructors which produce result
// objects.
func WithName(name string) ClientOption {
	return optionFunc(func(c *Client) {
		c.WithName(name)
	})
}

// WithPolicy is a ClientOption that specifies retriable policy.
//
//   retriable.New(retriable.WithPolicy(p))
//
// This option cannot be provided for constructors which produce result
// objects.
func WithPolicy(policy *Policy) ClientOption {
	return optionFunc(func(c *Client) {
		c.WithPolicy(policy)
	})
}

// WithTLS is a ClientOption that specifies TLS configuration.
//
//   retriable.New(retriable.WithTLS(t))
//
// This option cannot be provided for constructors which produce result
// objects.
func WithTLS(tlsConfig *tls.Config) ClientOption {
	return optionFunc(func(c *Client) {
		c.WithTLS(tlsConfig)
	})
}

// Client is custom implementation of http.Client
type Client struct {
	lock       sync.RWMutex
	httpClient *http.Client // Internal HTTP client.
	headers    map[string]string
	Name       string
	Policy     *Policy // Rery policy for http requests
}

// New creates a new Client
func New(opts ...ClientOption) *Client {
	c := &Client{
		Name: "retriable",
		httpClient: &http.Client{
			Timeout: time.Second * 30,
		},
		Policy: NewDefaultPolicy(),
	}

	for _, opt := range opts {
		opt.applyOption(c)
	}
	return c
}

// WithHeaders adds additional headers to the request
func (c *Client) WithHeaders(headers map[string]string) *Client {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.headers == nil {
		c.headers = map[string]string{}
	}

	for key, val := range headers {
		c.headers[key] = val
	}
	return c
}

// AddHeader adds additional header to the request
func (c *Client) AddHeader(header, value string) *Client {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.headers == nil {
		c.headers = map[string]string{}
	}

	c.headers[header] = value
	return c
}

// WithName modifies client's name for logging purposes.
func (c *Client) WithName(name string) *Client {
	c.lock.RLock()
	defer c.lock.RUnlock()
	c.Name = name
	return c
}

// WithPolicy modifies retriable policy.
func (c *Client) WithPolicy(policy *Policy) *Client {
	c.lock.RLock()
	defer c.lock.RUnlock()
	c.Policy = policy
	return c
}

// WithTLS modifies TLS configuration.
func (c *Client) WithTLS(tlsConfig *tls.Config) *Client {
	c.lock.RLock()
	defer c.lock.RUnlock()
	if tlsConfig != nil {
		tr := &http.Transport{
			TLSClientConfig:     tlsConfig,
			TLSHandshakeTimeout: time.Second * 3,
			IdleConnTimeout:     time.Hour,
			MaxIdleConnsPerHost: 2,
		}
		err := http2.ConfigureTransport(tr)
		if err != nil {
			logger.Errorf("api=WithTLS, err=[%s]", errors.ErrorStack(err))
		} else {
			c.httpClient.Transport = tr
		}
	} else {
		c.httpClient.Transport = nil
	}
	return c
}

// NewDefaultPolicy returns default policy
func NewDefaultPolicy() *Policy {
	return &Policy{
		Retries: map[int]ShouldRetry{
			// 0 is connection related
			0: shouldRetryFactory(5, time.Second*2, "connection"),
			// TooManyRequests (429) is returned when rate limit is exceeded
			http.StatusTooManyRequests: shouldRetryFactory(3, time.Second, "rate-limit"),
			// Unavailble (503) is returned when Cluster knows there's no leader or service is not ready yet
			http.StatusServiceUnavailable: shouldRetryFactory(10, time.Second/2, "unavailable"),
			// Bad Gateway (502) is returned when the node that gets the request thinks
			// there's a leader, but that node is down, treat them both as an election
			// in progress.
			http.StatusBadGateway: shouldRetryFactory(10, time.Second/2, "gateway"),
		},
		TotalRetryLimit: 10,
	}
}

// Get fetches the specified resource from the specified hosts
// The supplied hosts are tried in order until one succeeds.
// It will decode the response payload into the supplied
// body parameter.
// It returns the HTTP status code, and an optional error.
// For responses with status codes >= 300 it will try and convert the response
// into a Go error.
// If configured, this call will wait & retry on rate limit and leader election errors.
// path should be an absolute URI path, i.e. /foo/bar/baz
func (c *Client) Get(ctx context.Context, hosts []string, path string, body interface{}) (int, error) {
	resp, err := c.executeRequest(ctx, http.MethodGet, hosts, path, nil)
	if err != nil {
		return 0, errors.Trace(err)
	}
	defer resp.Body.Close()

	return c.DecodeResponse(resp, body)
}

// Delete removes the specified resource from the specified hosts.
// The supplied hosts are tried in order until one succeeds.
// It will decode the response payload into the supplied
// body parameter.
// It returns the HTTP status code, and an optional error.
// For responses with status codes >= 300 it will try and convert the response
// into a Go error.
// If configured, this call will wait & retry on rate limit and leader election errors.
// path should be an absolute URI path, i.e. /foo/bar/baz
func (c *Client) Delete(ctx context.Context, hosts []string, path string, body interface{}) (int, error) {
	resp, err := c.executeRequest(ctx, http.MethodDelete, hosts, path, nil)

	if err != nil {
		return 0, errors.Trace(err)
	}
	defer resp.Body.Close()

	return c.DecodeResponse(resp, body)
}

// GetResponse executes a GET request against the specified hosts,
// The supplied hosts are tried in order until one succeeds.
// It returns the HTTP status code, and an optional error.
// For responses with status codes >= 300 it will try and convert the response
// into a Go error.
// If configured, this call will wait & retry on rate limit and leader election errors.
// path should be an absolute URI path, i.e. /foo/bar/baz
func (c *Client) GetResponse(ctx context.Context, hosts []string, path string, body io.Writer) (int, error) {
	resp, err := c.executeRequest(ctx, http.MethodGet, hosts, path, nil)
	if err != nil {
		return 0, errors.Trace(err)
	}
	defer resp.Body.Close()

	return c.extractResponse(resp, body)
}

// PostBody makes POST request against the specified hosts.
// The supplied hosts are tried in order until one succeeds.
// It will decode the response payload into the supplied
// body parameter.
// It returns the HTTP status code, and an optional error.
// For responses with status codes >= 300 it will try and convert the response
// into a Go error.
// If configured, this call will wait & retry on rate limit and leader election errors.
// path should be an absolute URI path, i.e. /foo/bar/baz
func (c *Client) PostBody(ctx context.Context, hosts []string, path string, reqBody []byte, body interface{}) (int, error) {
	resp, err := c.executeRequest(ctx, http.MethodPost, hosts, path, reqBody)
	if err != nil {
		return 0, errors.Trace(err)
	}
	defer resp.Body.Close()

	return c.DecodeResponse(resp, body)
}

func (c *Client) executeRequest(ctx context.Context, httpMethod string, hosts []string, path string, reqBody []byte) (*http.Response, error) {
	var many *httperror.ManyError
	var err error
	var resp *http.Response
	for i, host := range hosts {
		resp, err = c.doHTTP(ctx, httpMethod, host, path, reqBody)
		if !c.shouldTryDifferentHost(resp, err) {
			break
		}

		// either success or error
		if err != nil {
			many = many.Add(host, err)
		} else if resp != nil {
			if resp.StatusCode >= 400 {
				many = many.Add(host, httperror.New(resp.StatusCode, "reques_failed", "%s %s %s %s",
					httpMethod, host, path, resp.Status))
			}
			// if not the last host, then close it
			if i < len(hosts)-1 {
				resp.Body.Close()
				resp = nil
			}
		}

		logger.Debugf("api=executeRequest, %s %s %s [%v]", httpMethod, host, path, many.Error())
	}

	if resp != nil {
		return resp, nil
	}

	return nil, many
}

// doHTTP wraps calling an HTTP method with retries.
// TODO: convert reqBody to reader
func (c *Client) doHTTP(ctx context.Context, httpMethod string, host string, path string, reqBody []byte) (*http.Response, error) {
	uri := host + path
	logger.Tracef("api=doHTTP, httpMethod='%s', host='%s', path='%s', uri='%s'", httpMethod, host, path, uri)

	var reader io.Reader
	if httpMethod == http.MethodPost && reqBody != nil {
		reader = bytes.NewReader(reqBody)
	}

	req, err := http.NewRequest(httpMethod, uri, reader)
	if err != nil {
		return nil, errors.Trace(err)
	}

	req = req.WithContext(ctx)

	for header, val := range c.headers {
		req.Header.Add(header, val)
	}

	return c.Do(req)
}

// Do wraps calling an HTTP method with retries.
func (c *Client) Do(r *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error
	var retries int

	req, err := convertRequest(r)
	if err != nil {
		return nil, errors.Trace(err)
	}

	for retries = 0; ; retries++ {
		// Always rewind the request body when non-nil.
		if req.body != nil {
			body, err := req.body()
			if err != nil {
				return resp, errors.Trace(err)
			}
			if c, ok := body.(io.ReadCloser); ok {
				req.Request.Body = c
			} else {
				req.Request.Body = ioutil.NopCloser(body)
			}
		}

		resp, err = c.httpClient.Do(req.Request)

		// Check if we should continue with retries.
		shouldRetry, sleepDuration, reason := c.Policy.ShouldRetry(req.Request, resp, err, retries)
		if !shouldRetry {
			break
		}

		desc := fmt.Sprintf("%s %s", req.Request.Method, req.Request.URL)
		if resp != nil {
			if resp.Status != "" {
				desc += " "
				desc += resp.Status
			}
			c.consumeResponseBody(resp)
		}

		logger.Warningf("api=Do, name=%s, retries=%d, description='%s', reason='%s', sleep=[%v]",
			c.Name, retries, desc, reason, sleepDuration.Seconds())
		time.Sleep(sleepDuration)
	}

	return resp, err
}

// shouldTryDifferentHost returns true if a connection error occurred
// or response has a specific HTTP status:
// - StatusInternalServerError
// - StatusServiceUnavailable
// - StatusGatewayTimeout
// - StatusTooManyRequests
// In that case, the caller should try to send the same request to a different host.
func (c *Client) shouldTryDifferentHost(resp *http.Response, err error) bool {
	if err != nil {
		return true
	}
	if resp == nil {
		return true
	}
	if resp.StatusCode == http.StatusInternalServerError ||
		resp.StatusCode == http.StatusServiceUnavailable ||
		resp.StatusCode == http.StatusGatewayTimeout ||
		resp.StatusCode == http.StatusTooManyRequests {
		return true
	}
	return false
}

// consumeResponseBody is a helper to safely consume the remaining response body
func (c *Client) consumeResponseBody(r *http.Response) {
	if r != nil && r.Body != nil {
		io.Copy(ioutil.Discard, r.Body)
	}
}

// DecodeResponse will look at the http response, and map it back to either
// the body parameters, or to an error
// [retrying rate limit errors should be done before this]
func (c *Client) DecodeResponse(resp *http.Response, body interface{}) (int, error) {
	if resp.StatusCode >= http.StatusMultipleChoices { // 300
		e := new(httperror.Error)
		e.HTTPStatus = resp.StatusCode
		bodyCopy := bytes.Buffer{}
		bodyTee := io.TeeReader(resp.Body, &bodyCopy)
		if err := json.NewDecoder(bodyTee).Decode(e); err != nil || e.Code == "" {
			io.Copy(ioutil.Discard, bodyTee) // ensure all of body is read
			// Unable to parse as Error, then return body as error
			return resp.StatusCode, errors.New(string(bodyCopy.Bytes()))
		}
		return resp.StatusCode, e
	}

	switch body.(type) {
	case io.Writer:
		_, err := io.Copy(body.(io.Writer), resp.Body)
		if err != nil {
			return resp.StatusCode, errors.Errorf("unable to read body response to (%T) type: %s", body, err.Error())
		}
	default:
		if err := json.NewDecoder(resp.Body).Decode(body); err != nil {
			return resp.StatusCode, errors.Errorf("unable to decode body response to (%T) type: %s", body, err.Error())
		}
	}

	return resp.StatusCode, nil
}

// extractResponse will look at the http response, and map it back to either
// the body parameters, or to an error
// [retrying rate limit errors should be done before this]
func (c *Client) extractResponse(resp *http.Response, body io.Writer) (int, error) {
	if resp.StatusCode >= http.StatusMultipleChoices { // 300
		e := new(httperror.Error)
		e.HTTPStatus = resp.StatusCode
		bodyCopy := bytes.Buffer{}
		bodyTee := io.TeeReader(resp.Body, &bodyCopy)
		if err := json.NewDecoder(bodyTee).Decode(e); err != nil || e.Code == "" {
			io.Copy(ioutil.Discard, bodyTee) // ensure all of body is read
			// Unable to parse as Error, then return body as error
			return resp.StatusCode, errors.New(string(bodyCopy.Bytes()))
		}
		return resp.StatusCode, e
	}

	if _, err := io.Copy(body, resp.Body); err != nil {
		return resp.StatusCode, errors.Errorf("unable to decode body response (%T) error: %s", body, err.Error())
	}

	return resp.StatusCode, nil
}

func shouldRetryFactory(limit int, wait time.Duration, reason string) ShouldRetry {
	return func(r *http.Request, resp *http.Response, err error, retries int) (bool, time.Duration, string) {
		return (limit >= retries), wait, reason
	}
}

var nonRetriableErrors = []string{
	"no such host",
	"TLS handshake error",
	"certificate signed by unknown authority",
	"client didn't provide a certificate",
	"tls: bad certificate",
	"server gave HTTP response to HTTPS client",
}

// ShouldRetry returns if connection should be retried
func (p *Policy) ShouldRetry(r *http.Request, resp *http.Response, err error, retries int) (bool, time.Duration, string) {
	if err != nil {
		errStr := err.Error()
		logger.Errorf("api=ShouldRetry, error_type=%T, err='%s'", err, errStr)

		select {
		// If the context is finished, don't bother processing the
		case <-r.Context().Done():
			return false, 0, Cancelled
		default:
		}

		if r.TLS != nil {
			logger.Errorf("api=ShouldRetry, complete=%t, mutual=%t, tls_peers=%d, tls_chains=%d",
				resp.TLS.HandshakeComplete,
				resp.TLS.NegotiatedProtocolIsMutual,
				len(resp.TLS.PeerCertificates),
				len(resp.TLS.VerifiedChains))
			for i, c := range resp.TLS.PeerCertificates {
				logger.Errorf("  [%d] CN: %s, Issuer: %s",
					i, c.Subject.CommonName, c.Issuer.CommonName)
			}
		}

		if slices.StringContainsOneOf(errStr, nonRetriableErrors) {
			return false, 0, NonRetriableError
		}

		// On error, use 0 code
		if fn, ok := p.Retries[0]; ok {
			return fn(r, resp, err, retries)
		}
		return false, 0, NonRetriableError
	}

	// Success codes 200-399
	if resp.StatusCode < 400 {
		return false, 0, Success
	}

	if resp.StatusCode == 404 {
		return false, 0, NotFound
	}

	if resp.StatusCode == 400 || resp.StatusCode == 401 {
		return false, 0, NonRetriableError
	}

	if retries >= p.TotalRetryLimit {
		return false, 0, LimitExceeded
	}

	if fn, ok := p.Retries[resp.StatusCode]; ok {
		return fn(r, resp, err, retries)
	}

	return false, 0, NonRetriableError
}
