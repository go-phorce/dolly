package xhttp

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-phorce/dolly/xlog"
	"github.com/stretchr/testify/assert"
)

const (
	//This DateFormat is meant to imitate
	logPrefixLength = len("2006-01-02 15:04:05.000000   | ")
)

func TestProfiler_CPUProfile(t *testing.T) {
	checkProfileGenerated(t, "?profile.cpu", ProfileCPU)
}

func TestProfiler_MemProfile(t *testing.T) {
	checkProfileGenerated(t, "?profile.mem", ProfileMem)
}

func TestProfiler_CPUAndMemProfile(t *testing.T) {
	checkProfileGenerated(t, "?profile.mem&profile.cpu", ProfileMem, ProfileCPU)
}

func TestProfiler_ProfileType(t *testing.T) {
	assert.Equal(t, "cpu", ProfileCPU.String(), "ProfileCPU")
	assert.Equal(t, "mem", ProfileMem.String(), "ProfileMem")
	bogus := ProfileType(42)
	assert.Equal(t, "<Unknown ProfileType>", bogus.String(), "ProfileType")
}

func TestProfiler_LogProfile(t *testing.T) {
	logdata := &bytes.Buffer{}
	writer := bufio.NewWriter(logdata)
	xlog.SetFormatter(xlog.NewPrettyFormatter(writer, false))

	lp := LogProfile()
	r, err := http.NewRequest(http.MethodGet, "/foo/bar", nil)
	if err != nil {
		t.Fatalf("Unable to create http.Request: %v", err)
	}
	lp(ProfileCPU, r, "/foo/cpu_123")
	result := logdata.String()[logPrefixLength:]
	assert.Equal(t, "xhttp: api=LogProfile, profile=cpu, status=created, url=/foo/bar, location=/foo/cpu_123\n", result)
}

func TestProfiler_Defaults(t *testing.T) {
	handler := testHandler{t, http.StatusOK, []byte("OK")}
	rph, err := NewRequestProfiler(&handler, "", nil, nil)
	ph := rph.(*requestProfiler)
	if err != nil {
		t.Fatalf("Error constructing RequestProfiler: %v", err)
	}
	if ph.allow == nil {
		t.Errorf("Expecting Allow Callback to be defaulted but is nil")
	}
	if ph.created == nil {
		t.Errorf("Expected Created callback to be defaulted but is nil")
	}
	if ph.dir == "" {
		t.Errorf("Expected dir to be defaulted to a tempdir, but is empty")
	}
}

func TestProfiler_CreateDir(t *testing.T) {
	dir, err := ioutil.TempDir("", "create_dir")
	if err != nil {
		t.Fatalf("Unable to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)
	dir = filepath.Join(dir, "profiles")
	handler := testHandler{t, http.StatusOK, []byte("OK")}
	_, err = NewRequestProfiler(&handler, dir, nil, nil)
	_, err = os.Stat(dir)
	if err != nil {
		t.Errorf("Dir wasn't created: %v", err)
	}
}

// consumeProfileType verifies that c is in the pt slice, and sets that index to 0
func consumeProfileType(t *testing.T, c ProfileType, pt []ProfileType) {
	for i, p := range pt {
		if p == c {
			pt[i] = 0
			return
		}
	}
	t.Errorf("Expected to find ProfileType %v in ProfileTypes %v", c, pt)
}

func checkProfileGenerated(t *testing.T, qs string, expectedProfileTypes ...ProfileType) {
	handler := testHandler{t, http.StatusOK, []byte("OK")}
	logdata := &bytes.Buffer{}
	writer := bufio.NewWriter(logdata)
	xlog.SetFormatter(xlog.NewPrettyFormatter(writer, false))

	profilesCreated := 0
	allowedCount := 0
	allowed := true
	allowedCb := func(pt ProfileType, req *http.Request) bool {
		allowedCount++
		if pt != ProfileCPU && pt != ProfileMem {
			t.Errorf("Allow callback received with invalid ProfileType %v", pt)
		}
		if req == nil {
			t.Errorf("Allow callback received with nil http.Request")
		}
		return allowed
	}
	ph, err := NewRequestProfiler(&handler, "", allowedCb, func(pt ProfileType, req *http.Request, file string) {
		profilesCreated++
		consumeProfileType(t, pt, expectedProfileTypes)
		if req.URL.Path != "/foo" {
			t.Errorf("Expected *http.Request seems wrong, expecting URI /foo, but got %s", req.URL.Path)
		}
		_, err := os.Stat(file)
		if err != nil {
			t.Errorf("Supplied profile file has an error: %v", err)
		}
		os.Remove(file)
	})
	if err != nil {
		t.Fatalf("Unexpected error created RequestProfiler handler: %v", err)
	}
	req, err := http.NewRequest(http.MethodGet, "/foo", nil)
	res := httptest.NewRecorder()
	if err != nil {
		t.Fatalf("Unexpected error created http.Request: %v", err)
	}
	ph.ServeHTTP(res, req)
	if profilesCreated != 0 {
		t.Errorf("Not expecting profile to be created, but %d were", profilesCreated)
	}
	assertRespEqual(t, res, http.StatusOK, "OK")
	req, err = http.NewRequest(http.MethodGet, "/foo"+qs, nil)
	res = httptest.NewRecorder()
	if err != nil {
		t.Fatalf("Unexpected error created http.Request: %v", err)
	}
	ph.ServeHTTP(res, req)
	if profilesCreated != len(expectedProfileTypes) {
		t.Errorf("Expecting profiles %d to be created, but got %d", len(expectedProfileTypes), profilesCreated)
	}
	if allowedCount != profilesCreated {
		t.Errorf("Expecting Allow callback to be called %d, but got %d", profilesCreated, allowedCount)
	}
	assertRespEqual(t, res, http.StatusOK, "OK")
	if logdata.Len() > 0 {
		t.Errorf("Unexpected log enties written\n%s", logdata.String())
	}
	allowedCount = 0
	profilesCreated = 0
	allowed = false
	res = httptest.NewRecorder()
	ph.ServeHTTP(res, req)
	if allowedCount != len(expectedProfileTypes) {
		t.Errorf("Expected allow callback to be called %d times, but got %d calls", len(expectedProfileTypes), allowedCount)
	}
	if profilesCreated > 0 {
		t.Errorf("Allow callback said not to create the profile, but %d profiles were created", profilesCreated)
	}
}
