package cf_test

import (
	"testing"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"

	"code.cloudfoundry.org/lager/v3/lagertest"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
)

func TestLeveledLoggerAdapter_Error(t *testing.T) {
	RegisterTestingT(t)
	logger := lagertest.NewTestLogger("test")
	log := cf.LeveledLoggerAdapter{Logger: logger}
	log.Error("Error message", "key1", "value1", "key2", "value2", "key3", "value3")
	Eventually(logger.Buffer).Should(Say(`{"timestamp":"[0-9]+.[0-9]+","source":"test","message":"test.Error message","log_level":2,"data":{"key1":"value1","key2":"value2","key3":"value3"}}`))
}

func TestLeveledLoggerAdapter_Info(t *testing.T) {
	RegisterTestingT(t)
	logger := lagertest.NewTestLogger("test")
	log := cf.LeveledLoggerAdapter{Logger: logger}
	log.Info("Info message", "key1", "value1", "key2", "value2", "key3", "value3")
	Eventually(logger.Buffer).Should(Say(`{"timestamp":"[0-9]+.[0-9]+","source":"test","message":"test.Info message","log_level":1,"data":{"key1":"value1","key2":"value2","key3":"value3"}}`))
}

func TestLeveledLoggerAdapter_Debug(t *testing.T) {
	RegisterTestingT(t)
	logger := lagertest.NewTestLogger("test")
	log := cf.LeveledLoggerAdapter{Logger: logger}
	log.Debug("Debug message", "key1", "value1", "key2", "value2", "key3", "value3")
	Eventually(logger.Buffer).Should(Say(`{"timestamp":"[0-9]+.[0-9]+","source":"test","message":"test.Debug message","log_level":0,"data":{"key1":"value1","key2":"value2","key3":"value3"}}`))
}

func TestLeveledLoggerAdapter_Warn(t *testing.T) {
	RegisterTestingT(t)
	logger := lagertest.NewTestLogger("test")
	log := cf.LeveledLoggerAdapter{Logger: logger}
	log.Warn("a message", "key1", "value1", "key2", "value2", "key3", "value3")
	Eventually(logger.Buffer).Should(Say(`{"timestamp":"[0-9]+.[0-9]+","source":"test","message":"test.Warning: a message","log_level":1,"data":{"key1":"value1","key2":"value2","key3":"value3"}}`))
}

func TestLeveledLoggerAdapter_NoLastValue(t *testing.T) {
	RegisterTestingT(t)
	logger := lagertest.NewTestLogger("test")
	log := cf.LeveledLoggerAdapter{Logger: logger}
	log.Warn("a message", "key1", "value1", "key2", "value2", "key3")
	Eventually(logger.Buffer).Should(Say(`{"timestamp":"[0-9]+.[0-9]+","source":"test","message":"test.Warning: a message","log_level":1,"data":{"key1":"value1","key2":"value2"}}`))
}
