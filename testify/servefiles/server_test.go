package servefiles

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type serverTestSuite struct {
	suite.Suite
	ft *fakeT
	s  *Server
}

func Test_Server(t *testing.T) {
	suite.Run(t, new(serverTestSuite))
}

func (s *serverTestSuite) SetupTest() {
	s.ft = new(fakeT)
	s.s = New(s.ft, "testdata/primary", "testdata/base")
}

func (s *serverTestSuite) TearDownTest() {
	s.s.Close()
}

func (s *serverTestSuite) doHTTPCall(method, path string, body io.Reader, expStatusCode int) []byte {
	req, err := http.NewRequest(method, s.s.URL()+path, body)
	s.Require().NoError(err)
	res, err := http.DefaultClient.Do(req)
	s.Require().NoError(err)
	defer res.Body.Close()
	resBody, err := ioutil.ReadAll(res.Body)
	s.Require().NoError(err)
	s.Require().Equal(expStatusCode, res.StatusCode)
	return resBody
}

func (s *serverTestSuite) Test_Default() {
	resp := s.doHTTPCall(http.MethodGet, "/def", nil, http.StatusOK)
	s.JSONEq(`{"def":true}`, string(resp))
	s.Equal(1, s.s.RequestCount("/def"))
	s.Equal(map[string]int{"/def": 1}, s.s.RequestCounts())
}

func (s *serverTestSuite) Test_WithStatusCode() {
	resp := s.doHTTPCall(http.MethodGet, "/withCode", nil, http.StatusBadRequest)
	s.JSONEq(`{"code":"BOOM"}`, string(resp))
	s.Equal(1, s.s.RequestCount("/withCode"))
	s.Equal(map[string]int{"/withCode": 1}, s.s.RequestCounts())
}

func (s *serverTestSuite) Test_Sequence() {
	s.testSequence("/withSeq", http.StatusOK)
}

func (s *serverTestSuite) Test_SequenceWithCode() {
	s.testSequence("/withSeqAndCode", http.StatusCreated)
}

func (s *serverTestSuite) testSequence(reqPath string, expStatusCode int) {
	resp := s.doHTTPCall(http.MethodGet, reqPath, nil, expStatusCode)
	s.JSONEq(`{"seq":1}`, string(resp))
	s.Equal(1, s.s.RequestCount(reqPath))
	resp = s.doHTTPCall(http.MethodGet, reqPath, nil, expStatusCode)
	s.JSONEq(`{"seq":2}`, string(resp))
	s.Equal(2, s.s.RequestCount(reqPath))
	s.Equal(map[string]int{reqPath: 2}, s.s.RequestCounts())
	// run off the end of supplied sequences, get a 404
	resp = s.doHTTPCall(http.MethodGet, reqPath, nil, http.StatusNotFound)
}

func (s *serverTestSuite) Test_Token() {
	resp := s.doHTTPCall(http.MethodGet, "/services/oauth2/token", nil, http.StatusOK)
	var d map[string]interface{}
	s.Require().NoError(json.Unmarshal(resp, &d))
	s.Equal(s.s.URL(), d["instance_url"])
	s.Equal(s.s.URL()+"/id/00DT0000000DpvcMAC/005B0000001JwvAIAS", d["id"])
	s.EqualValues(1459975184111, d["issued_at"])
}

func (s *serverTestSuite) Test_NotAuthToken() {
	// verify that something that looks like the auth token, but is at a different
	// url is not modified.
	resp := s.doHTTPCall(http.MethodGet, "/services/not_oauth/token", nil, http.StatusOK)
	var d map[string]interface{}
	s.Require().NoError(json.Unmarshal(resp, &d))
	s.Equal("https://login.acme.com/id/00DT0000000DpvcMAC/005B0000001JwvAIAS", d["id"])
	s.Equal("https://na1.acme.com/", d["instance_url"])
	s.EqualValues(1459975184111, d["issued_at"])
}

func (s *serverTestSuite) Test_LastRequestBody() {
	reqBody := `{"hello":"world"}`
	resp := s.doHTTPCall(http.MethodPost, "/def", bytes.NewBufferString(reqBody), http.StatusOK)
	s.Equal(reqBody, string(s.s.LastBody("/def")))
	s.JSONEq(`{"def":true}`, string(resp))
	s.Equal(1, s.s.RequestCount("/def"))

	reqBody = `{"hello":"world2"}`
	resp = s.doHTTPCall(http.MethodPut, "/def", bytes.NewBufferString(reqBody), http.StatusOK)
	s.Equal(reqBody, string(s.s.LastBody("/def")))
	s.JSONEq(`{"def":true}`, string(resp))
	s.Equal(2, s.s.RequestCount("/def"))
}

func (s *serverTestSuite) Test_Missing() {
	resp := s.doHTTPCall(http.MethodGet, "/missing", nil, http.StatusNotFound)
	s.JSONEq(`[{"errorCode": "NOT_FOUND", "message": "The requested resource does not exist"}]`, string(resp))
}

func (s *serverTestSuite) Test_SequencedStatusCodes() {
	s.doHTTPCall(http.MethodGet, "/statusCodes", nil, http.StatusBadRequest)
	s.doHTTPCall(http.MethodGet, "/statusCodes", nil, http.StatusNotFound)
	s.doHTTPCall(http.MethodGet, "/statusCodes", nil, http.StatusOK)
}

func Test_StatusCode(t *testing.T) {
	r := requestSettings{}
	assert.Equal(t, 200, r.statusCode(1))
	r.StatusCode = 200
	assert.Equal(t, 200, r.statusCode(1))
	r.StatusCode = 404
	assert.Equal(t, 404, r.statusCode(1))
	r = requestSettings{
		StatusCodes: []int{400, 500, 201},
	}
	assert.Equal(t, 400, r.statusCode(1))
	assert.Equal(t, 500, r.statusCode(2))
	assert.Equal(t, 201, r.statusCode(3))
	assert.Equal(t, 200, r.statusCode(4))
}

func Test_HandleAuthFixup(t *testing.T) {
	src := `{"instance_url": "https://login.acme.com", "id":"https://na1.acme.com/1/2", "sig":1234}`
	dest := &bytes.Buffer{}
	handleAuthFixup("http://127.0.0.1:1234", dest, bytes.NewBufferString(src))
	exp := `{"instance_url": "http://127.0.0.1:1234", "id":"http://127.0.0.1:1234/1/2", "sig":1234}`
	assert.JSONEq(t, dest.String(), exp)
}

func Test_NotFound(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/foo", nil)
	require.NoError(t, err)
	req.RequestURI = "/foo"
	res := httptest.NewRecorder()
	ft := new(fakeT)
	s := Server{t: ft}
	s.notFound(res, req)
	assert.Equal(t, http.StatusNotFound, res.Code)
	assert.JSONEq(t, `[{"errorCode": "NOT_FOUND", "message": "The requested resource does not exist"}]`, res.Body.String())
	if assert.True(t, len(ft.messages) > 0) {
		assert.Equal(t, ft.messages[0], "No response file exists for /foo, returning a 404 response")
	}
}

// a MinimalTestingT impl that captures info about calls to it, outside of our tests actual testing.T
type fakeT struct {
	messages []string
	failed   bool
}

func (f *fakeT) Logf(format string, args ...interface{}) {
	f.messages = append(f.messages, fmt.Sprintf(format, args...))
}

func (f *fakeT) Errorf(format string, args ...interface{}) {
	f.Logf(format, args...)
	f.failed = true
}

func (f *fakeT) FailNow() {
	f.failed = true
}
