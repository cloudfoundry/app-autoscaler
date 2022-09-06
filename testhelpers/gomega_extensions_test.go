package testhelpers_test

import (
	"net/http"
	"testing"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"

	"github.com/stretchr/testify/assert"
)

func TestRespondWithMultiple_rollsCorrectly(t *testing.T) {
	value := -1
	handlers := testhelpers.RespondWithMultiple(
		func(resp http.ResponseWriter, req *http.Request) { value = 1 },
		func(resp http.ResponseWriter, req *http.Request) { value = 2 },
		func(resp http.ResponseWriter, req *http.Request) { value = 3 },
		func(resp http.ResponseWriter, req *http.Request) { value = 4 },
		func(resp http.ResponseWriter, req *http.Request) { value = 5 },
	)

	handlers(nil, nil)
	assert.Equal(t, 1, value)
	handlers(nil, nil)
	assert.Equal(t, 2, value)
	handlers(nil, nil)
	assert.Equal(t, 3, value)
	handlers(nil, nil)
	assert.Equal(t, 4, value)
	handlers(nil, nil)
	assert.Equal(t, 5, value)
	handlers(nil, nil)
	assert.Equal(t, 5, value)
}

func TestRespondWithMultiple_empty(t *testing.T) {
	handlers := testhelpers.RespondWithMultiple()

	handlers(nil, nil)
}

func TestRoundRobinWithMultiple_rollsCorrectly(t *testing.T) {
	value := -1
	handlers := testhelpers.RoundRobinWithMultiple(
		func(resp http.ResponseWriter, req *http.Request) { value = 1 },
		func(resp http.ResponseWriter, req *http.Request) { value = 2 },
		func(resp http.ResponseWriter, req *http.Request) { value = 3 },
		func(resp http.ResponseWriter, req *http.Request) { value = 4 },
		func(resp http.ResponseWriter, req *http.Request) { value = 5 },
	)

	handlers(nil, nil)
	assert.Equal(t, 1, value)
	handlers(nil, nil)
	assert.Equal(t, 2, value)
	handlers(nil, nil)
	assert.Equal(t, 3, value)
	handlers(nil, nil)
	assert.Equal(t, 4, value)
	handlers(nil, nil)
	assert.Equal(t, 5, value)
	handlers(nil, nil)
	assert.Equal(t, 1, value)
}

func TestRoundRobinWithMultiple_empty(t *testing.T) {
	handlers := testhelpers.RoundRobinWithMultiple()

	handlers(nil, nil)
}
