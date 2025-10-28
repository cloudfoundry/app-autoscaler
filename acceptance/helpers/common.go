package helpers

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
)

func getNames(resources []cfResource) []string {
	var names []string
	for _, item := range resources {
		names = append(names, item.Name)
	}
	return names
}

func filterByPrefix(prefix string, in []string) []string {
	var filtered []string
	for _, item := range in {
		if strings.HasPrefix(item, prefix) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func FailOnError(err error) {
	if err != nil {
		Fail(err.Error())
	}
}
