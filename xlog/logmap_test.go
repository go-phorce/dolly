package xlog_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-phorce/dolly/xlog"
)

func Test_LevelTrace(t *testing.T) {
	assert.True(t, xlog.INFO < xlog.TRACE)
	assert.True(t, xlog.TRACE < xlog.DEBUG)
}

func Test_LevelChar(t *testing.T) {
	assert.Equal(t, xlog.CRITICAL.Char(), "C")
	assert.Equal(t, xlog.ERROR.Char(), "E")
	assert.Equal(t, xlog.WARNING.Char(), "W")
	assert.Equal(t, xlog.NOTICE.Char(), "N")
	assert.Equal(t, xlog.INFO.Char(), "I")
	assert.Equal(t, xlog.TRACE.Char(), "T")
	assert.Equal(t, xlog.DEBUG.Char(), "D")
}

func Test_LevelString(t *testing.T) {
	assert.Equal(t, xlog.CRITICAL.String(), "CRITICAL")
	assert.Equal(t, xlog.ERROR.String(), "ERROR")
	assert.Equal(t, xlog.WARNING.String(), "WARNING")
	assert.Equal(t, xlog.NOTICE.String(), "NOTICE")
	assert.Equal(t, xlog.INFO.String(), "INFO")
	assert.Equal(t, xlog.TRACE.String(), "TRACE")
	assert.Equal(t, xlog.DEBUG.String(), "DEBUG")
}

func Test_ParseLevel(t *testing.T) {
	tcases := []struct {
		name  string
		level xlog.LogLevel
		err   string
	}{
		{"CRITICAL", xlog.CRITICAL, ""},
		{"C", xlog.CRITICAL, ""},
		{"ERROR", xlog.ERROR, ""},
		{"E", xlog.ERROR, ""},
		{"0", xlog.ERROR, ""},
		{"WARNING", xlog.WARNING, ""},
		{"W", xlog.WARNING, ""},
		{"1", xlog.WARNING, ""},
		{"NOTICE", xlog.NOTICE, ""},
		{"N", xlog.NOTICE, ""},
		{"2", xlog.NOTICE, ""},
		{"INFO", xlog.INFO, ""},
		{"I", xlog.INFO, ""},
		{"3", xlog.INFO, ""},
		{"TRACE", xlog.TRACE, ""},
		{"T", xlog.TRACE, ""},
		{"4", xlog.TRACE, ""},
		{"DEBUG", xlog.DEBUG, ""},
		{"D", xlog.DEBUG, ""},
		{"5", xlog.DEBUG, ""},
		{"w", xlog.CRITICAL, "unable to parse log level: w"},
	}

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			l, err := xlog.ParseLevel(tc.name)
			if tc.err != "" {
				assert.Equal(t, xlog.CRITICAL, l)
				if assert.Error(t, err) {
					assert.Equal(t, tc.err, err.Error())
				}
			} else {
				assert.Equal(t, tc.level, l)
			}
		})
	}
}

func Test_SetLevel(t *testing.T) {
	l := new(xlog.LogLevel)
	err := l.Set("W")
	require.NoError(t, err)
	assert.Equal(t, xlog.WARNING, *l)
}

func Test_GetRepoLogger(t *testing.T) {
	r, err := xlog.GetRepoLogger("repo1")
	require.Error(t, err)
	assert.Equal(t, "no packages registered for repo: repo1", err.Error())

	logger1 := xlog.NewPackageLogger("repo1", "pkg1")
	r, err = xlog.GetRepoLogger("repo1")
	require.NoError(t, err)
	logger1.Println("repo1", "pkg1")

	logger2 := xlog.NewPackageLogger("repo2", "pkg2")
	xlog.NewPackageLogger("repo2", "pkg3")
	r = xlog.MustRepoLogger("repo2")
	r.SetLogLevel(map[string]xlog.LogLevel{"pkg2": xlog.DEBUG})
	logger2.Println("repo1", "pkg1")

	mm, err := r.ParseLogLevelConfig("pkg2=N,pkg3=DEBUG")
	require.NoError(t, err)
	assert.Equal(t, xlog.NOTICE, mm["pkg2"])
	assert.Equal(t, xlog.DEBUG, mm["pkg3"])
}
