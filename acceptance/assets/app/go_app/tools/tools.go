//go:build tools

package tools

import (
	_ "github.com/maxbrunsfeld/counterfeiter/v6"
	_ "github.com/ogen-go/ogen"
)

// This file imports packages that are used when running go generate,
// but not otherwise depended on by built code.
