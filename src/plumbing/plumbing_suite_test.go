package plumbing_test

import (
	"io/ioutil"
	"log"
	"net"
	"testing"

	"google.golang.org/grpc/grpclog"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestPlumbing(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	grpclog.SetLogger(log.New(ioutil.Discard, "", log.LstdFlags))
	log.SetOutput(GinkgoWriter)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plumbing Suite")
}

func startListener(addr string) net.Listener {
	var lis net.Listener
	f := func() error {
		var err error
		lis, err = net.Listen("tcp", addr)
		return err
	}
	Eventually(f).Should(Succeed())

	return lis
}
