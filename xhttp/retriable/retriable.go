package retriable

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-phorce/dolly/algorithms/slices"
	"github.com/go-phorce/dolly/xhttp/httperror"
	"github.com/go-phorce/dolly/xlog"
	"github.com/juju/errors"
	"golang.org/x/net/http2"
)

var logger = xlog.NewPackageLogger("github.com/go-phorce/dolly/xhttp", "retry")

// Context represents interface for the HTTP request context
type Context interface {
	SetHeaders(r *http.Request)
}

// lenReader is an interface implemented by many in-memory io.Reader's. Used
// for automatically sending the right Content-Length header when possible.
type lenReader interface {
	Len() int
}

//Requestor defines interface to make HTTP calls
type Requestor interface {
	Do(r *http.Request) (*http.Response, error)
}

// ReaderFunc is the type of function that can be given natively to NewRequest
type ReaderFunc func() (io.Reader, error)

// Request wraps the metadata needed to create HTTP requests.
type Request struct {
	// body is a seekable reader over the request body payload. This is
	// used to rewind the request data in between retries.
	body ReaderFunc

	// Embed an HTTP request directly. This makes a *Request act exactly
	// like an *http.Request so that all meta methods are supported.
	*http.Request
}

// NewRequest creates a new wrapped request.
func NewRequest(method, url string, rawBody io.ReadSeeker) (*Request, error) {
	var body ReaderFunc
	var contentLength int64

	if rawBody != nil {
		raw := rawBody.(io.ReadSeeker)
		body = func() (io.Reader, error) {
			raw.Seek(0, 0)
			return ioutil.NopCloser(raw), nil
		}
		if lr, ok := raw.(lenReader); ok {
			contentLength = int64(lr.Len())
		}
	}

	httpReq, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, errors.Trace(err)
	}
	httpReq.ContentLength = contentLength

	return &Request{body: body, Request: httpReq}, nil
}

// convertRequest wraps http.Request into retry.Request
func convertRequest(req *http.Request) (*Request, error) {
	var body io.ReadSeeker
	if req != nil && req.Body != nil {
		defer req.Body.Close()
		bodyBytes, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return nil, errors.Trace(err)
		}
		body = bytes.NewReader(bodyBytes)
	}

	r, err := NewRequest(req.Method, req.URL.String(), body)
	if err != nil {
		return nil, errors.Trace(err)
	}
	r.Request = r.WithContext(req.Context())
	for header, vals := range req.Header {
		for _, val := range vals {
			r.Request.Header.Add(header, val)
		}
	}

	return r, nil
}

// ShouldRetry specifies a policy for handling retries. It is called
// following each request with the response, error values returned by
// the http.Client and the number of already made retries.
// If ShouldRetry returns false, the Client stops retrying
// and returns the response to the caller. The
// Client will close any response body when retrying, but if the retry is
// aborted it is up to the caller to properly close any response body before returning.
type ShouldRetry func(resp *http.Response, err error, retries int) (bool, time.Duration, string)

// Policy represents the retry policy
type Policy struct {

	// Retries specifies a map of HTTP Status code to ShouldRetry function,
	// 0 status code indicates a connection related error (network, TLS, DNS etc.)
	Retries map[int]ShouldRetry

	// Maximum number of retries.
	TotalRetryLimit int

	// TotalRetryTimeout specifies a timeout period during which connection should succeed
	TotalRetryTimeout time.Duration
}

// Client is custom implementation of http.Client
type Client struct {
	lock        sync.RWMutex
	name        string       // Name of the client.
	httpClient  *http.Client // Internal HTTP client.
	RetryPolicy *Policy      // Rery policy for http requests
}

// New creates a new Client
func New(name string, tlsClientConfig *tls.Config) (*Client, error) {
	var tr *http.Transport
	if tlsClientConfig != nil {
		tr = &http.Transport{
			TLSClientConfig:     tlsClientConfig,
			TLSHandshakeTimeout: time.Second * 3,
			IdleConnTimeout:     time.Hour,
			MaxIdleConnsPerHost: 2,
		}
		err := http2.ConfigureTransport(tr)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}
	c := &Client{
		name: name,
		httpClient: &http.Client{
			Transport: tr,
			Timeout:   time.Minute,
		},
		RetryPolicy: NewDefaultPolicy(),
	}

	return c, nil
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
		TotalRetryLimit:   10,
		TotalRetryTimeout: time.Second * 10,
	}
}

// Get fetches the specified resource from the specified hosts[the supplied hosts are
// tried in order until one succeeds, or we run out]
// It will decode the response payload into the supplied
// body parameter. it returns the HTTP status code, and an optional error
// for responses with status codes >= 300 it will try and convert the response
// into an go error.
// If configured, this call will wait & retry on rate limit and leader election errors
// path should be an absolute URI path, i.e. /foo/bar/baz
func (c *Client) Get(ctx Context, hosts []string, path string, body interface{}) (int, error) {
	resp, err := c.executeRequest(ctx, http.MethodGet, hosts, path, nil, 0)
	if err != nil {
		return 0, errors.Trace(err)
	}
	defer resp.Body.Close()

	return c.DecodeResponse(resp, body)
}

// Delete removes the specified resource from the specified hosts[the supplied hosts are
// tried in order until one succeeds, or we run out]
// It will decode the response payload into the supplied
// body parameter. it returns the HTTP status code, and an optional error
// for responses with status codes >= 300 it will try and convert the response
// into an go error.
// If configured, this call will wait & retry on rate limit and leader election errors
// path should be an absolute URI path, i.e. /foo/bar/baz
func (c *Client) Delete(ctx Context, hosts []string, path string, body interface{}) (int, error) {
	resp, err := c.executeRequest(ctx, http.MethodDelete, hosts, path, nil, 0)

	if err != nil {
		return 0, errors.Trace(err)
	}
	defer resp.Body.Close()

	return c.DecodeResponse(resp, body)
}

// GetResponse executes a GET request against the specified hosts[the supplied hosts are
// tried in order until one succeeds, or we run out].
// Path should be an absolute URI path, i.e. /foo/bar/baz
// the resulting HTTP body will be returned into the supplied body parameter, and the
// http status code returned.
func (c *Client) GetResponse(ctx Context, hosts []string, path string, body io.Writer) (int, error) {
	resp, err := c.executeRequest(ctx, http.MethodGet, hosts, path, nil, 0)
	if err != nil {
		return 0, errors.Trace(err)
	}
	defer resp.Body.Close()

	return c.extractResponse(ctx, resp, body)
}

// PostBody makes POST request against the specified hosts[the supplied hosts are
// tried in order until one succeeds, or we run out]
// each host should include all the protocol/host/port preamble, e.g. http://foo.bar:3444
// path should be an absolute URI path, i.e. /foo/bar/baz
// if set, the callers identity will be passed to the server via the X-Identity header
func (c *Client) PostBody(ctx Context, hosts []string, path string, reqBody []byte, body interface{}) (int, error) {
	resp, err := c.executeRequest(ctx, http.MethodPost, hosts, path, reqBody, 0)
	if err != nil {
		return 0, errors.Trace(err)
	}
	defer resp.Body.Close()

	return c.DecodeResponse(resp, body)
}

func (c *Client) executeRequest(ctx Context, httpMethod string, hosts []string, path string, reqBody []byte, retriesOnError int) (*http.Response, error) {
	var err error
	var resp *http.Response
	for _, host := range hosts {
		resp, err = c.doHTTP(ctx, httpMethod, host, path, reqBody)
		if c.shouldTryDifferentHost(resp, err) {
			if err != nil {
				logger.Errorf("api=executeRequest, httpMethod='%s', host='%s', path='%s', err=[%v]",
					httpMethod, host, path, errors.ErrorStack(err))
			} else {
				logger.Errorf("api=executeRequest, httpMethod='%s', host='%s', path='%s', status=%v",
					httpMethod, host, path, resp.Status)
			}
			continue
		}

		return resp, errors.Trace(err)
	}
	/*
		if fn, ok := c.RetryPolicy.Retries[0]; ok {
			if shouldRetry, sleepDuration, reason := fn(resp, err, retriesOnError); shouldRetry {
				logger.Tracef("api=executeRequest, httpMethod=%s, hosts=[%s], path='%s', retriesOnError=%d, sleepDuration=[%v], reason='%s'",
					httpMethod, strings.Join(hosts, ","), path, retriesOnError, sleepDuration.Round(1*time.Millisecond), reason)

				time.Sleep(sleepDuration)

				return c.executeRequest(ctx, httpMethod, hosts, path, reqBody, retriesOnError+1)
			}
		}
	*/
	if err != nil {
		return nil, errors.Annotatef(err, "api=executeRequest, status=failed, hosts=[%s], retriesOnError=%d", strings.Join(hosts, ","), retriesOnError)
	}
	return nil, errors.Errorf("api=executeRequest, status=failed, hosts=[%s], retriesOnError=%d", strings.Join(hosts, ","), retriesOnError)
}

// doHTTP wraps calling an HTTP method with retries.
func (c *Client) doHTTP(ctx Context, httpMethod string, host string, path string, reqBody []byte) (*http.Response, error) {
	uri := host + path
	logger.Tracef("api=doHTTP, httpMethod='%s', host='%s', path='%s', uri='%s'", httpMethod, host, path, uri)

	var reader io.Reader
	if httpMethod == http.MethodPost {
		reader = bytes.NewReader(reqBody)
	}

	req, err := http.NewRequest(httpMethod, uri, reader)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if ctx != nil {
		ctx.SetHeaders(req)
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
		shouldRetry, sleepDuration, reason := c.RetryPolicy.ShouldRetry(resp, err, retries)
		if !shouldRetry {
			break
		}

		desc := fmt.Sprintf("%s %s", req.Request.Method, req.Request.URL)
		if err == nil {
			desc = fmt.Sprintf("%s (status: %d)", desc, resp.StatusCode)
			c.consumeResponseBody(resp)
		}

		logger.Infof("api=Do, name=%s, retries=%d, description='%s', reason='%s', sleepDuration=[%v]",
			c.name, retries, desc, reason, sleepDuration.Seconds())
		time.Sleep(sleepDuration)
	}

	return resp, err
}

// shouldTryDifferentHost returns true if a connection error occurred
// or response is internal server error.
// In that case, the caller should try to send the same request to a different host.
func (c *Client) shouldTryDifferentHost(resp *http.Response, err error) bool {
	if err != nil {
		return true
	}
	if resp == nil {
		return true
	}
	if resp.StatusCode == http.StatusInternalServerError {
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
		if err := json.NewDecoder(bodyTee).Decode(e); err != nil {
			io.Copy(ioutil.Discard, bodyTee) // ensure all of body is read
			return resp.StatusCode, errors.Errorf("HTTP StatusCode %d: [unable to parse body: %v] Body:%s", resp.StatusCode, err, bodyCopy.Bytes())
		}
		if e.Code != "" {
			return resp.StatusCode, e
		}
		// expecting { "code" : "fooo", "message":"bar"}, but didn't get that, so just return the entire body structure with an error
		return resp.StatusCode, errors.Errorf("HTTP StatusCode %d: Body: %s", resp.StatusCode, bodyCopy.Bytes())
	}

	switch body.(type) {
	case io.Writer:
		_, err := io.Copy(body.(io.Writer), resp.Body)
		if err != nil {
			return resp.StatusCode, errors.Errorf("unable to read body response to (%T) type: %v", body, err)
		}
	default:
		if err := json.NewDecoder(resp.Body).Decode(body); err != nil {
			return resp.StatusCode, errors.Errorf("unable to decode body response to (%T) type: %v", body, err)
		}
	}

	return resp.StatusCode, nil
}

// extractResponse will look at the http response, and map it back to either
// the body parameters, or to an error
// [retrying rate limit errors should be done before this]
func (c *Client) extractResponse(ctx Context, resp *http.Response, body io.Writer) (int, error) {
	if resp.StatusCode >= http.StatusMultipleChoices { // 300
		e := new(httperror.Error)
		e.HTTPStatus = resp.StatusCode
		bodyCopy := bytes.Buffer{}
		bodyTee := io.TeeReader(resp.Body, &bodyCopy)
		if err := json.NewDecoder(bodyTee).Decode(e); err != nil {
			io.Copy(ioutil.Discard, bodyTee) // ensure all of body is read
			return resp.StatusCode, errors.Errorf("HTTP StatusCode %d: [unable to parse body: %v] Body:%s", resp.StatusCode, err, bodyCopy.Bytes())
		}
		if e.Code != "" {
			return resp.StatusCode, e
		}
		// expecting { "code" : "fooo", "message":"bar"}, but didn't get that, so just return the entire body structure with an error
		return resp.StatusCode, errors.Errorf("HTTP StatusCode %d: Body: %s", resp.StatusCode, bodyCopy.Bytes())
	}

	if _, err := io.Copy(body, resp.Body); err != nil {
		return resp.StatusCode, errors.Errorf("unable to decode body response (%T) error: %v", body, err)
	}

	return resp.StatusCode, nil
}

// Timeout return timeout duration for connection retries
func (c *Client) Timeout() time.Duration {
	return c.RetryPolicy.TotalRetryTimeout
}

// SetRetryPolicy changes the retry policy
func (c *Client) SetRetryPolicy(r *Policy) *Client {
	c.lock.RLock()
	defer c.lock.RUnlock()
	c.RetryPolicy = r
	return c
}

func shouldRetryFactory(limit int, wait time.Duration, reason string) ShouldRetry {
	return func(resp *http.Response, err error, retries int) (bool, time.Duration, string) {
		return (limit >= retries), wait, reason
	}
}

var nonRetriableErrors = []string{
	"TLS handshake error",
	"certificate signed by unknown authority",
}

const nonretriablereason = "non-retriable error"

// ShouldRetry returns if connection should be retried
func (p *Policy) ShouldRetry(resp *http.Response, err error, retries int) (bool, time.Duration, string) {
	if err != nil {
		errStr := err.Error()
		logger.Errorf("api=ShouldRetry, error_type=%T, err='%s'", err, errStr)

		if slices.StringContainsOneOf(errStr, nonRetriableErrors) {
			return false, 0, nonretriablereason
		}

		// On error, use 0 code
		if fn, ok := p.Retries[0]; ok {
			return fn(resp, err, retries)
		}
		return false, 0, ""
	}

	// Success codes 200-399
	if resp.StatusCode < 400 {
		return false, 0, "success"
	}

	if retries >= p.TotalRetryLimit {
		return false, 0, "retry-limit-exceeded"
	}

	if fn, ok := p.Retries[resp.StatusCode]; ok {
		return fn(resp, err, retries)
	}

	return false, 0, nonretriablereason
}
