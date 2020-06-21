package version

import (
	"fmt"
)

// Info describes a version of an executable
type Info struct {
	Major  uint   `json:"major"`
	Minor  uint   `json:"minor"`
	Commit uint   `json:"commi"`
	Build  string `json:"build"`
	flt    float32
}

// PopulateFromBuild will parse the major/minor values from the build string
// the build string is expected to be in the format
// major.minor-commit
// and can be populated from git using
// 	GIT_VERSION := $(shell git describe --dirty --always --tags --long)
// and then using gofmt to substitute it into a template
func (v *Info) PopulateFromBuild() {
	fmt.Sscanf(v.Build, "v%d.%d.%d", &v.Major, &v.Minor, &v.Commit)
	fmt.Sscanf(v.Build, "v%f-", &v.flt)
	v.flt = v.flt*1000000 + float32(v.Commit)
}

func (v Info) String() string {
	return v.Build
}

// GreaterOrEqual returns true if the version 'v' is the same or new that the supplied parameter 'other'
// This only examines the Major & Minor field (as the SHA in Build provides no ordering indication)
func (v Info) GreaterOrEqual(than Info) bool {
	if v.Major > than.Major {
		return true
	}
	if v.Major < than.Major {
		return false
	}
	return v.Minor >= than.Minor
}

// Float returns the version Major/Minor as a float Major.Minor
// e.g. given Major:3 Minor:52001, it'll return 3.52001
// this is only valid if PopulateFromBuild has been called.
func (v Info) Float() float32 {
	return v.flt
}
