package testhelpers

import "io/ioutil"

func LoadFile(filename string) string {
	file, err := ioutil.ReadFile(filename)
	FailOnError("Could not read file", err)
	return string(file)
}
