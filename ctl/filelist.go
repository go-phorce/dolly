package ctl

import "strings"

// FilesList allows to specify files in command
type FilesList []string

// Set value
func (d *FilesList) Set(value string) error {
	*d = append(*d, value)
	return nil
}

// IsCumulative always returns true
func (d *FilesList) IsCumulative() bool {
	return true
}

// String returns coma separated list of files
func (d *FilesList) String() string {
	return strings.Join(*d, ",")
}
