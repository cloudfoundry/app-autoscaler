//go:build tools

package main

import (
	_ "github.com/maxbrunsfeld/counterfeiter/v6"
)

// This file imports packages that are used when running go generate,
// but not otherwise depended on by built code.
