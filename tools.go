// Package tools for go mod

// +build tools

package tools

import (
	_ "github.com/cloudflare/cfssl/cmd/cfssl"
	_ "github.com/cloudflare/cfssl/cmd/cfssljson"
	_ "github.com/go-phorce/cov-report/cmd/cov-report"
	_ "github.com/jteeuwen/go-bindata"
	_ "github.com/mattn/goveralls"
	_ "github.com/stretchr/testify"
	_ "golang.org/x/lint/golint"
	_ "golang.org/x/tools/cmd/godoc"
	_ "golang.org/x/tools/cmd/gorename"
	_ "golang.org/x/tools/cmd/guru"
	_ "golang.org/x/tools/cmd/stringer"
)
