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
		// fmt.Errorf("Could not read file \"%s\"\nError: %w", filePath, err)
		panic(`Assumed guid-schema not found.
This is a programming-error as the file must be on the hardcorded location.`)
	}

	type Schema struct {
		Schemas struct {
			Guid struct {
				Pattern string `json:"pattern"`
			} `json:"guid"`
		} `json:"schemas"`
	}
	var schema Schema
	json.Unmarshal(jsonData, schema)
	pattern := schema.Schemas.Guid.Pattern

	r, err := regexp.CompilePOSIX(pattern)
	if err != nil {
		panic(`The provided pattern is invalid.
This is a programming-error as the pattern must be a valid POSIX-regexp.`)
	}

	return CfGuidParser{regexp: r}
}

func (p CfGuidParser) Parse(rawGuid string) (CfGuid, error) {
	matched := p.regexp.MatchString(rawGuid) // regexp.MatchString(p.regexp, rawGuid)
	if ! matched {
		msg := fmt.Sprintf("The provided string does not look like a Cloud Foundry GUID: %s", rawGuid)
		return "<guid-parsing-error>", errors.New(msg)
	}

	return CfGuid(rawGuid), nil
}

func ParseGuid(rawGuid string) (CfGuid, error) {
	p := NewCfGuidParser()
	return p.Parse(rawGuid)
}
