package log

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

const auditLogFilename = "audit_test.log"

type testSource int

const (
	srcFoo testSource = iota
	srcBar
)

func (i testSource) ID() int {
	return int(i)
}

func (i testSource) String() string {
	return "src" + strconv.Itoa(int(i))
}

type testEventType int

const (
	evtBar testEventType = iota
	evtFoo
)

func (i testEventType) ID() int {
	return int(i)
}

func (i testEventType) String() string {
	return "type" + strconv.Itoa(int(i))
}

// Logger contains information about the configuration of a logger/log rotation
type logger struct {
	// Directory contains where to store the log files; if value is empty, them stderr is used for output
	Directory string
	// MaxAgeDays controls how old files are before deletion
	MaxAgeDays int
	// MaxSizeMb contols how large a single log file can be before its rotated
	MaxSizeMb int
}

func Test_FileAuditor(t *testing.T) {
	suite.Run(t, new(FileTestSuite))
}

type FileTestSuite struct {
	suite.Suite
	cfg logger
}

func (f *FileTestSuite) SetupTest() {
	dir, err := ioutil.TempDir("", "fileaudit")
	f.Require().NoError(err)
	f.cfg = logger{
		Directory:  dir,
		MaxSizeMb:  10,
		MaxAgeDays: 1,
	}
}

func (f *FileTestSuite) TearDownTest() {
	os.RemoveAll(f.cfg.Directory)
}

func (f *FileTestSuite) Test_Event() {
	fa, err := New(auditLogFilename, f.cfg.Directory, f.cfg.MaxAgeDays, f.cfg.MaxSizeMb)
	f.Require().NoError(err)
	fa.Audit(srcBar.String(), evtFoo.String(), "rt/bob1-1", "1234-2345-3456", 55556, fmt.Sprintf("%s:%s", "KernelModule", "HASH:123"))
	fa.Close()
	log, err := ioutil.ReadFile(filepath.Join(f.cfg.Directory, auditLogFilename))
	f.Require().NoError(err)
	s := string(log)
	f.True(strings.Contains(s, "src1:type1:rt/bob1-1:1234-2345-3456:55556:KernelModule:HASH:123"), "Didn't find expected log entry, log is\n%s", s)
}
