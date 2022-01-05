package metricsgateway_test

import (
	"io/ioutil"
	"log"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc/grpclog"

	"testing"
)

func TestMetricsgateway(t *testing.T) {
	grpclog.SetLoggerV2(grpclog.NewLoggerV2(GinkgoWriter, ioutil.Discard, ioutil.Discard))
	log.SetOutput(GinkgoWriter)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Metricsgateway Suite")
}
