package xhttp

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-phorce/dolly/xlog"
)

const (
	//This DateFormat is meant to imitate
	prefixLength = len("2006-01-02 15:04:05.000000   | ")
)

func assertRespEqual(t *testing.T, res *httptest.ResponseRecorder, expStatusCode int, expBody string) {
	if expStatusCode != res.Code {
		t.Errorf("Expecting statusCode %d, but got %d", expStatusCode, res.Code)
	}
	if expBody != res.Body.String() {
		t.Errorf("Expecting responseBody '%s', but got '%s'", expBody, res.Body.String())
	}
}

func TestHttp_ResponseCapture(t *testing.T) {
	w := httptest.NewRecorder()
	rc := NewResponseCapture(w)
	var rw http.ResponseWriter = rc // ensure rc can be used as a ResponseWriter
	rw.Header().Add("Content-Type", "text/plain")
	rw.WriteHeader(http.StatusNotFound)
	body := []byte("/foo not found")
	rw.Write(body)
	rw.Write(body) // write this 2 to ensure we're accumulate bytes written
	if rc.StatusCode() != http.StatusNotFound {
		t.Errorf("ResponseCapture didn't report the expected status code set by the caller, got %d", rc.StatusCode())
	}
	expBodyLen := uint64(len(body)) * 2
	if rc.BodySize() != expBodyLen {
		t.Errorf("Expected BodySize to be %d, but was %d", expBodyLen, rc.BodySize())
	}
	// check that it actually passed onto the delegate ResponseWriter
	assertRespEqual(t, w, http.StatusNotFound, "/foo not found/foo not found")
}

type testHandler struct {
	t            *testing.T
	statusCode   int
	responseBody []byte
}

func (th *testHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/foo" {
		th.t.Errorf("ultimate handler didn't see correct request")
	}
	w.WriteHeader(th.statusCode)
	w.Write(th.responseBody)
}

func TestHttp_RequestLoggerNullHandler(t *testing.T) {
	defer func() {
		err := recover()
		if err == nil {
			t.Fatalf("Expected panic but didn't get one")
		}
		if err != errNoHandler {
			t.Errorf("Unexpected panic value of %v, expecting %v", err, errNoHandler)
		}
	}()
	NewRequestLogger(nil, "BOB", nil, time.Millisecond, "git.soma.salesforce.com/raphty/pkg/xhttp-test")
}

func TestHttp_RequestLogger(t *testing.T) {
	testResponseBody := []byte(`Hello World`)
	handler := &testHandler{t, http.StatusBadRequest, testResponseBody}
	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/foo", nil)
	testRemoteIP := "127.0.0.1"
	testRemotePort := "51500"
	r.RemoteAddr = fmt.Sprintf("%v:%v", testRemoteIP, testRemotePort)
	r.ProtoMajor = 1
	r.ProtoMinor = 1

	tw := bytes.Buffer{}
	writer := bufio.NewWriter(&tw)
	xlog.SetFormatter(xlog.NewPrettyFormatter(writer, false))

	testLogPrefix := "BOB"
	logHandler := NewRequestLogger(handler, testLogPrefix, nil, time.Millisecond, "git.soma.salesforce.com/raphty/pkg/xhttp")
	logHandler.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Status code set by ultimate handler wasn't returned by the logging wrapper")
	}
	if tw.Len() == 0 {
		t.Fatalf("A request was processed, but nothing was logged")
	}

	logLine := tw.String()[prefixLength:]
	logParts := strings.Split(logLine, ":")
	const something = "<SOMETHING>"
	// API, HTTP Method, ClientUser, Path, IP Address, Port, HTTP status code, HTTP version, response length, duration, agent
	exp := []string{"xhttp", " " + testLogPrefix, "GET", "", "/foo", testRemoteIP, testRemotePort, strconv.Itoa(http.StatusBadRequest), "1.1", strconv.Itoa(len(testResponseBody)), something, "no-agent\n"}
	if len(logParts) != len(exp) {
		t.Errorf("Expecting %d log line items, but got %d (%q)", len(exp), len(logParts), logParts)
	}
	for i, part := range exp {
		if i >= len(logParts) {
			t.Fatalf("Expecting an item at index %d but there wasn't one, log: %s", i, logLine)
		}
		if part != something && part != logParts[i] {
			t.Errorf("Expecting '%v' for log part %d, but got '%v', log: %s", part, i, logParts[i], logLine)
		}
		if part == something && len(logParts[i]) == 0 {
			t.Errorf("Expecting some kind of value for part %d, but got an empty string, log: %s", i, logLine)
		}
	}
}

func makeExtractor(t *testing.T) AdditionalLogExtractor {
	return func(w *ResponseCapture, r *http.Request) []string {
		if w == nil {
			t.Fatalf("LogExtractor passed nil ResponseCatpure")
		}
		if r == nil {
			t.Fatalf("LogExtractor passed nil Request")
		}
		return []string{"JIVE", "TURKEY"}
	}
}

func TestHttp_RequestLoggerWithExtractor(t *testing.T) {
	handler := &testHandler{t, http.StatusOK, []byte(`Hello World`)}
	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/foo", nil)
	tw := bytes.Buffer{}
	writer := bufio.NewWriter(&tw)
	xlog.SetFormatter(xlog.NewPrettyFormatter(writer, false))
	lg := NewRequestLogger(handler, "ASD", makeExtractor(t), time.Millisecond, "")
	lg.ServeHTTP(w, r)
	logLine := tw.String()[prefixLength:]
	if !strings.HasSuffix(logLine, ":JIVE:TURKEY\n") {
		t.Errorf("Log Line should end with our custom extracted values, but was '%v'", logLine)
	}
}
