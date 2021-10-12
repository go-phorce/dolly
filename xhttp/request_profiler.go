package xhttp

import (
	"io/ioutil"
	h "net/http"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"

	"github.com/pkg/errors"
)

// ProfileType indicates the type of profile created
type ProfileType int

const (
	// ProfileCPU indicates a CPU Usage profile
	ProfileCPU ProfileType = 1
	// ProfileMem indicates a memory usage profile
	ProfileMem ProfileType = 2
)

func (p ProfileType) String() string {
	switch p {
	case ProfileCPU:
		return "cpu"
	case ProfileMem:
		return "mem"
	default:
		return "<Unknown ProfileType>"
	}
}

// AllowProfiling defines a callback function that is called to see if requested
// profile(s) are allowed for this request [this allows you to implement auth
// or some other kind of finer grained profiling permissions]
type AllowProfiling func(types ProfileType, r *h.Request) bool

// ProfileCreated defines a callback geneated when a new profile has been created
// You can supplied a custom callback, or use the prebuilt logging callback
type ProfileCreated func(t ProfileType, r *h.Request, f string)

// LogProfile returns a ProfileCreated callback function that writes the details
// of the generated profile to the supplied logger
func LogProfile() ProfileCreated {
	return func(t ProfileType, r *h.Request, f string) {
		logger.Infof("profile=%v, status=created, url=%s, location=%s", t, r.URL.Path, f)
	}
}

// NewRequestProfiler wraps the supplied delegate and can enable cpu or memory
// profiling of a specific request based on the presence of ?profile.cpu or
// ?profile.mem in the request query string
// the allow function if supplied, is given the chance decide if the given
// request should be profiled, this allows users of this middleware to decide
// what if any access policy they want WRT to generating profiles. if allow
// is nil, all indicated requests will be profiled.
//
// Note that go doesn't allow for concurrent profiles, so all requests that
// are to be profiled are serialized. [unprofiled requests are unaffected]
//
// if dir is "" then a temp dir is created to contain the profiler outputfiles
// otherwise the indicated directory is used.
//
// details of each generated profile are passed to the supplied createdCallback function
func NewRequestProfiler(delegate h.Handler, dir string, allow AllowProfiling, createdCallback ProfileCreated) (h.Handler, error) {
	var err error
	if dir == "" {
		dir, err = ioutil.TempDir("", "request_profiler")
		if err != nil {
			return nil, errors.WithStack(err)
		}
	} else {
		if err := os.MkdirAll(dir, 0600); err != nil {
			return nil, errors.WithStack(err)
		}
	}
	if allow == nil {
		allow = allowAny
	}
	if createdCallback == nil {
		createdCallback = noopCreated
	}
	return &requestProfiler{
		delegate: delegate,
		dir:      dir,
		allow:    allow,
		created:  createdCallback,
	}, nil
}

type requestProfiler struct {
	delegate    h.Handler
	dir         string
	profileLock sync.Mutex
	allow       AllowProfiling
	created     ProfileCreated
}

func allowAny(_ ProfileType, _ *h.Request) bool {
	return true
}

func noopCreated(_ ProfileType, _ *h.Request, f string) {
}

func (rp *requestProfiler) ServeHTTP(w h.ResponseWriter, r *h.Request) {
	qs := r.URL.Query()
	_, cpu := qs["profile.cpu"]
	_, mem := qs["profile.mem"]
	if cpu && rp.allow(ProfileCPU, r) {
		cpuf, err := ioutil.TempFile(rp.dir, "cpu_")
		if err != nil {
			logger.Errorf("status=unable_create_file, profile=cpu, err=%v", err)
		} else {
			rp.profileLock.Lock()
			if err = pprof.StartCPUProfile(cpuf); err != nil {
				rp.profileLock.Unlock()
				logger.Infof("status=unable_start, profile=cpu, err=%v", err)
				cpuf.Close()
				os.Remove(cpuf.Name())
			} else {
				defer func() {
					pprof.StopCPUProfile()
					rp.profileLock.Unlock()
					cpuf.Close()
					rp.created(ProfileCPU, r, cpuf.Name())
				}()
			}
		}
	}
	if mem && rp.allow(ProfileMem, r) {
		memf, err := ioutil.TempFile(rp.dir, "mem_")
		if err != nil {
			logger.Infof("status=unable_create_file, profile=memory, err=%v", err)
		} else {
			defer func() {
				runtime.GC()
				pprof.WriteHeapProfile(memf)
				memf.Close()
				rp.created(ProfileMem, r, memf.Name())
			}()
		}
	}

	rp.delegate.ServeHTTP(w, r)
}
