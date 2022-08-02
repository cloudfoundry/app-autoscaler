package testhelpers

import "io/ioutil"

func LoadFile(filename string) string {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		file, err = ioutil.ReadFile("testdata/" + filename)
	}
	FailOnError("Could not read file", err)
	return string(file)
}
