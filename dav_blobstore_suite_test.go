package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

func TestDavBlobstore(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "dav blobstore Suite")
}

var davServerPath string
var _ = SynchronizedBeforeSuite(func() []byte {
	server, err := gexec.Build("github.com/sykesm/dav-blobstore", "-race")
	Expect(err).NotTo(HaveOccurred())

	return []byte(server)
}, func(payload []byte) {
	davServerPath = string(payload)
})

var _ = SynchronizedAfterSuite(func() {
}, func() {
	gexec.CleanupBuildArtifacts()
})
