//go:build tools

package main

import (
	_ "github.com/apache/thrift/lib/go/thrift" // pin v0.23.0 to satisfy Black Duck CVE policy; tidy removes indirect pins without a direct import
	_ "github.com/maxbrunsfeld/counterfeiter/v6"
)

// This file imports packages that are used when running go generate,
// but not otherwise depended on by built code.
