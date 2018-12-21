// Package servefiles provides a way to mock a HTTP server endpoint
// by providing response payloads from the disk
package servefiles

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-phorce/dolly/xhttp/header"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MinimalTestingT defines the small subset of testing.T that we use
// having this be an interface makes plugging in other test runs and
// writing our own tests easier
type MinimalTestingT interface {
	require.TestingT
	Logf(format string, args ...interface{})
}

// Server implements a test server that response based on sets of pre-canned
// response contained in a directory supplied by calling SetBaseDir
type Server struct {
	t      MinimalTestingT
	server *httptest.Server

	lock       sync.Mutex
	baseDirs   []string
	reqCounts  map[string]int
	reqFiles   map[string]requestSettings
	reqHdrs    map[string]map[string][]string
	lastBodies map[string][]byte
}

// New creates a new instance of the test HTTP server.
// Once created you should call SetBaseDir to indicate where the fake responses
// are stored. You should call Close on the returned object when you're finished using it
// to shutdown the server
// if the supplied baseDirs is not empty, then SetBaseDirs will be called for you
// to configure the response mappings
func New(t MinimalTestingT, baseDirs ...string) *Server {
	s := &Server{
		reqCounts:  make(map[string]int),
		lastBodies: make(map[string][]byte),
		reqHdrs:    make(map[string]map[string][]string),
		t:          t,
	}
	if len(baseDirs) > 0 {
		s.SetBaseDirs(baseDirs...)
	}
	s.server = httptest.NewServer(s)
	return s
}

// SetBaseDirs updates the baseDir to a new directory, if the directory is not
// valid, it'll fail the test. The directory is expected to contain a requests.json
// file that provides the mapping from request URI to file with the response
// data in it. e.g.
// {
//    "/v1/status" : "status",
//    "get:/v1/status" : "getstatus"
// }
// the actual file used will be status.json, if this doesn't exist
// then it'll look for a file based on the number of requests to this URI
// e.g. status.1.json, status.2.json, etc.
// if there's no matching entry, or the specified file doesn't exist a
// 404 response is returned.
// the auth endpoint at /services/oauth2/token has special handling to update
// instance_url to the url of this test server [you still need to provide the
// base response file]
// requests.json must be in the first directory provided, named reponse payloaded
// will be used in the order of the supplied directoies, e.g. if requests.json
// says a response is describe.json, then it'll look in baseDirs[0] and if it can't
// find it there, look in baseDirs[1] and so on through the supplied list of baseDirs.
// This makes its easier for your tests to not have duplicated sets of response files
// by having fallback directories with common responses in.
// newBaseDirs must incldue at least one directory.
func (s *Server) SetBaseDirs(newBaseDirs ...string) {
	require.True(s.t, len(newBaseDirs) > 0, "Must supplied at least one directory in SetBaseDirs")
	for _, d := range newBaseDirs {
		stat, err := os.Stat(d)
		require.NoError(s.t, err, "Supplied baseDir %s should be a directory", d)
		require.True(s.t, stat.IsDir(), "Supplied baseDir %s should be a directory", d)
	}
	var reqMap map[string]requestSettings
	bytes, err := ioutil.ReadFile(filepath.Join(newBaseDirs[0], "requests.json"))
	require.NoError(s.t, err)
	require.NoError(s.t, json.Unmarshal(bytes, &reqMap))
	s.lock.Lock()
	defer s.lock.Unlock()
	s.reqFiles = reqMap
	s.baseDirs = append([]string(nil), newBaseDirs...)
}

type requestSettings struct {
	ContentType string            `json:"contentType"`
	Filename    string            `json:"file"`
	StatusCode  int               `json:"statusCode"`
	StatusCodes []int             `json:"statusCodes"`
	Headers     map[string]string `json:"headers"`
}

func (r *requestSettings) statusCode(reqCount int) int {
	if r.StatusCode == 0 {
		if reqCount <= len(r.StatusCodes) {
			return r.StatusCodes[reqCount-1]
		}
		return http.StatusOK
	}
	return r.StatusCode
}

// Close shuts down the test server
func (s *Server) Close() {
	s.server.Close()
}

// URL returns the base URL to this server
func (s *Server) URL() string {
	return s.server.URL
}

// RequestCounts returns a map of the number of requests processed for each URI
func (s *Server) RequestCounts() map[string]int {
	s.lock.Lock()
	defer s.lock.Unlock()
	c := make(map[string]int, len(s.reqCounts))
	for k, v := range s.reqCounts {
		c[k] = v
	}
	return c
}

// RequestCount returns the number of request processed for the indicated URI
func (s *Server) RequestCount(uri string) int {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.reqCounts[uri]
}

// LastBody returns the most recently recevied POST/PUT request body for the indicated URI
func (s *Server) LastBody(uri string) []byte {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.lastBodies[uri]
}

// LastReqHdr returns the headers of last request
func (s *Server) LastReqHdr(uri string) map[string][]string {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.reqHdrs[uri]
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestURI := r.RequestURI
	if requestURI == "" {
		requestURI = r.URL.Path
		if r.URL.RawQuery != "" {
			requestURI += "?" + r.URL.RawQuery
		}
	}
	s.t.Logf("sfdctest.Server got request for %s", requestURI)

	verb := strings.ToLower(r.Method)
	// first, try "get:/v1/resource"
	respInfo, exists := s.reqFiles[verb+":"+requestURI]
	if !exists {
		// then, try "/v1/resource"
		respInfo, exists = s.reqFiles[requestURI]
	}

	fileExt := ".json"
	if exists && respInfo.ContentType != "" {
		w.Header().Set(header.ContentType, respInfo.ContentType)
		switch respInfo.ContentType {
		case header.TextPlain:
			fileExt = ".txt"
		case header.ApplicationTimestampQuery:
			fileExt = ".tsq"
		case header.ApplicationTimestampReply:
			fileExt = ".tsr"
		default:
			fileExt = ".json"
		}
	} else {
		w.Header().Set(header.ContentType, header.ApplicationJSON)
	}

	var reqBody []byte
	var err error
	hasBody := false
	if r.Method == http.MethodPost || r.Method == http.MethodPut {
		reqBody, err = ioutil.ReadAll(r.Body)
		assert.NoError(s.t, err, "Unable to read request body for request to %s", requestURI)
		hasBody = true
	}
	s.lock.Lock()
	baseDirs := append([]string(nil), s.baseDirs...)
	count := s.reqCounts[requestURI]
	count++
	s.reqCounts[requestURI] = count
	if r.Header != nil {
		s.reqHdrs[requestURI] = r.Header
	}
	if hasBody {
		s.lastBodies[requestURI] = reqBody
	}
	s.lock.Unlock()
	if !exists {
		s.notFound(w, r)
		return
	}
	var f *os.File
	for _, baseDir := range baseDirs {
		fn := filepath.Join(baseDir, respInfo.Filename)
		if f, err = os.Open(fn + fileExt); err != nil {
			fnNext := fmt.Sprintf("%s.%d", fn, count)
			if f, err = os.Open(fnNext + fileExt); err != nil {
				// try original without extension
				if f, err = os.Open(fn); err != nil {
					continue
				}
			}
		}
		break
	}
	if f == nil {
		s.notFound(w, r)
		return
	}
	defer f.Close()
	s.t.Logf("Using %q to handle request to %q", f.Name(), requestURI)
	for header, val := range respInfo.Headers {
		w.Header().Set(header, val)
	}

	w.WriteHeader(respInfo.statusCode(count))
	if strings.HasPrefix(requestURI, "/services/oauth2/token") {
		handleAuthFixup(s.server.URL, w, f)
		return
	}
	io.Copy(w, f)
}

// handleAuthFixup is special handling for the token auth response which contains
// a URL back to the server, we need to update this with the specific URL for this
// test server
func handleAuthFixup(serverURL string, w io.Writer, f io.Reader) {
	var auth map[string]interface{}
	dec := json.NewDecoder(f)
	dec.UseNumber()
	dec.Decode(&auth)
	auth["instance_url"] = serverURL
	if id, exists := auth["id"]; exists {
		idURL, err1 := url.Parse(id.(string))
		testURL, err2 := url.Parse(serverURL)
		if err1 == nil && err2 == nil {
			idURL.Host = testURL.Host
			idURL.Scheme = testURL.Scheme
			auth["id"] = idURL.String()
		}
	}
	json.NewEncoder(w).Encode(&auth)
}

func (s *Server) notFound(w http.ResponseWriter, r *http.Request) {
	s.t.Logf("No response file exists for %s, returning a 404 response", r.URL.Path)
	notFound := `[{"errorCode": "NOT_FOUND", "message": "The requested resource does not exist"}]`
	w.WriteHeader(http.StatusNotFound)
	io.WriteString(w, notFound)
}
