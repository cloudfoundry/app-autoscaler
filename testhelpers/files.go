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

func BytesToFile(b []byte) string {
	if len(b) == 0 {
		return ""
	}

	file, err := os.CreateTemp("", "")
	FailOnError("Could create file", err)
	_, err = file.Write(b)
	FailOnError("Could write file", err)
	return file.Name()
}
