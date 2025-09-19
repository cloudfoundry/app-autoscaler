package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"time"
)

type ScalingType int
type ScalingStatus int

const (
	ScalingTypeDynamic ScalingType = iota
	ScalingTypeSchedule
)

const (
	ScalingStatusSucceeded ScalingStatus = iota
	ScalingStatusFailed
	ScalingStatusIgnored
)

const (
	AppStatusStopped = "STOPPED"
	AppStatusStarted = "STARTED"
)

type AppScalingHistory struct {
	AppId        string        `json:"app_id"`
	Timestamp    int64         `json:"timestamp"`
	ScalingType  ScalingType   `json:"scaling_type"`
	Status       ScalingStatus `json:"status"`
	OldInstances int           `json:"old_instances"`
	NewInstances int           `json:"new_instances"`
	Reason       string        `json:"reason"`
	Message      string        `json:"message"`
	Error        string        `json:"error"`
}

type AppMonitor struct {
	AppId      string
	MetricType string
	StatWindow time.Duration
}

type AppScalingResult struct {
	AppId             string        `json:"app_id"`
	Status            ScalingStatus `json:"status"`
	Adjustment        int           `json:"adjustment"`
	CooldownExpiredAt int64         `json:"cool_down_expired_at"`
}

// 🚧 To-do: Bring this in line with the content of “models/common_typges.go”.
// ================================================================================
// GUIDs
// ================================================================================

// Globally unique identifier in the context of a “Cloud Foundry” installation;
type CfGuid string

func (g CfGuid) String() string {
	return string(g)
}

type CfGuidParser struct {
	regexp *regexp.Regexp
}

func NewCfGuidParser() CfGuidParser {
	filePath := "../../../schema/json/shared_definitions.json"
	jsonData, err := os.ReadFile(filePath)
	if err != nil {
		errMsg := fmt.Errorf("could not read file \"%s\"\nError: %w", filePath, err)
		// Panicing here is O.K. because it does not make sense to launch autoscaler because it has
		// been shipped with an invalid (or non-existent) file that should contain a json-schema.
		panic(fmt.Sprintf(
			"%s\nThis is a programming-error as the file must be on the hardcoded location.",
			errMsg))
	}

	// Unfortunately there is no ordinary library comparable to
	// e.g. <https://crates.io/crates/serde_json> that allows to parse arbitrary JSON without
	// defining homomorphic structs in the host-language. So here comes a type that describes the
	// structure of the file `…/shared_definitions.json` (see above) with the sole intention to read
	// out the content of the field `pattern`.
	type Schema struct {
		Schemas struct {
			Guid struct {
				Pattern string `json:"pattern"`
			} `json:"guid"`
		} `json:"schemas"`
	}
	var schema Schema
	err = json.Unmarshal(jsonData, &schema)
	if err != nil {
		errMsg := fmt.Errorf("could not unmarshal JSON from file \"%s\"\nError: %w", filePath, err)
		// Panicing here is O.K. because it does not make sense to launch autoscaler because it has
		// been shipped with an file with invalid json-code.
		panic(fmt.Sprintf(
			"%s\nThis is a programming-error as the local Schema-struct must match roughly the file-structure.",
			errMsg))
	}
	pattern := schema.Schemas.Guid.Pattern

	r, err := regexp.CompilePOSIX(pattern)
	if err != nil {
		// Panicing here is O.K. because it does not make sense to launch autoscaler because it has
		// been shipped with an invalid json-schema.
		panic(`The provided pattern is invalid.
This is a programming-error as the pattern must be a valid POSIX-regexp.`)
	}

	return CfGuidParser{regexp: r}
}

func (p CfGuidParser) Parse(rawGuid string) (CfGuid, error) {
	matched := p.regexp.MatchString(rawGuid)
	if !matched {
		msg := fmt.Sprintf("The provided string does not look like a Cloud Foundry GUID: %s", rawGuid)
		return "<guid-parsing-error>", errors.New(msg)
	}

	return CfGuid(rawGuid), nil
}

func ParseGuid(rawGuid string) (CfGuid, error) {
	p := NewCfGuidParser()
	return p.Parse(rawGuid)
}
