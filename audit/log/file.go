// Package log provides an implmentation of Auditer that writes to local disk files
package log

import (
	"log"
	"os"
	"path/filepath"

	"github.com/go-phorce/pkg/audit"
	"github.com/juju/errors"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

// New return a new instance of an Auditor that writes audit entries to a local log file
func New(fileprefix, directory string, maxAgeDays int, maxSizeMb int) (audit.Auditor, error) {
	if err := os.MkdirAll(directory, 0755); err != nil {
		return nil, errors.Trace(err)
	}
	res := fileAuditor{
		fileWriter: lumberjack.Logger{
			Filename: filepath.Join(directory, fileprefix),
			MaxAge:   maxAgeDays,
			MaxSize:  maxSizeMb,
		},
	}
	res.logger = log.New(&res.fileWriter, "", log.Ldate|log.Ltime|log.LUTC)
	return &res, nil
}

type fileAuditor struct {
	fileWriter lumberjack.Logger
	logger     *log.Logger
}

func (f *fileAuditor) Close() error {
	return f.fileWriter.Close()
}

// Event logs event in the following format:
// {source}:{type}:{identity}:{contextID}:{raftIndex}:{message}
func (f *fileAuditor) Audit(
	source string,
	eventType string,
	identity string,
	contextID string,
	raftIndex uint64,
	message string) {
	f.logger.Printf("%s:%s:%s:%s:%d:%s\n",
		source, eventType, identity, contextID, raftIndex, message)
}
