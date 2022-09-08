package testhelpers

import (
	"os"
)

func LoadFile(filename string) string {
	file, err := os.ReadFile(filename)
	if err != nil {
		file, err = os.ReadFile("testdata/" + filename)
	}
	FailOnError("Could not read file", err)
	return string(file)
}
