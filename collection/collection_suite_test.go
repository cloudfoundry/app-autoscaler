package collection_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCollection(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Collection Suite")
}

type TestTSD struct {
	timestamp int64
	labels    map[string]string
}

func (t TestTSD) GetTimestamp() int64 {
	return t.timestamp
}

func (t TestTSD) HasLabels(labels map[string]string) bool {
	for k, v := range labels {
		if t.labels[k] == v {
			continue
		}
		return false
	}
	return true
}
