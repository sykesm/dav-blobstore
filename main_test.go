package main_test

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/sykesm/dav-blobstore"
)

var _ = Describe("main", func() {
	var (
		listenAddress  string
		u              *url.URL
		tempDir        string
		configFilePath string
		serverConfig   *main.Config

		session *gexec.Session
	)

	BeforeEach(func() {
		var err error

		listenAddress = fmt.Sprintf("127.0.0.1:%d", 14000+GinkgoParallelNode())
		u, err = url.Parse(fmt.Sprintf("http://%s/config.json", listenAddress))
		Expect(err).NotTo(HaveOccurred())

		tempDir, err = ioutil.TempDir("", "dav-blobstore")
		Expect(err).NotTo(HaveOccurred())

		serverConfig = &main.Config{
			BlobsPath:  tempDir,
			PublicRead: true,
		}

		configFilePath = filepath.Join(tempDir, "config.json")
		marshalToFile(configFilePath, serverConfig)
	})

	JustBeforeEach(func() {
		command := exec.Command(
			davServerPath,
			"--configFile", configFilePath,
			"--listenAddress", listenAddress,
		)

		var err error
		session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if session != nil {
			session.Kill()
			Eventually(session).Should(gexec.Exit())
		}
	})

	It("serves the configured blobs path over http", func() {
		Eventually(dial("tcp", listenAddress)).Should(Succeed())

		configFileContents, err := ioutil.ReadFile(configFilePath)
		Expect(err).NotTo(HaveOccurred())

		resp, err := http.Get(u.String())
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())

		Expect(body).To(BeEquivalentTo(configFileContents))
	})

	It("requires authentication for write requests", func() {
		Eventually(dial("tcp", listenAddress)).Should(Succeed())

		req, err := http.NewRequest(http.MethodDelete, u.String(), nil)
		Expect(err).NotTo(HaveOccurred())

		resp, err := http.DefaultClient.Do(req)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
	})

	Context("when public read is disabled", func() {
		BeforeEach(func() {
			serverConfig.PublicRead = false
			serverConfig.Users = map[string]string{
				"user": "password",
			}
			marshalToFile(configFilePath, serverConfig)
		})

		It("requires authentication for read requests", func() {
			Eventually(dial("tcp", listenAddress)).Should(Succeed())

			resp, err := http.Get(u.String())
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
		})

		It("allows read requests by authenticated users", func() {
			Eventually(dial("tcp", listenAddress)).Should(Succeed())

			u.User = url.UserPassword("user", "password")

			resp, err := http.Get(u.String())
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})
	})

	Context("when the configuration file cannot be opened", func() {
		BeforeEach(func() {
			err := os.Remove(configFilePath)
			Expect(err).NotTo(HaveOccurred())
		})

		It("fails with an error message", func() {
			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err).To(gbytes.Say("failed to load config data"))
		})
	})

	Context("when the configuration fails to load", func() {
		BeforeEach(func() {
			err := ioutil.WriteFile(configFilePath, []byte("!!invalid-json!!"), 0644)
			Expect(err).NotTo(HaveOccurred())
		})

		It("fails with an error message", func() {
			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err).To(gbytes.Say("failed to load config data"))
		})
	})

	Context("when the blobs path is missing from the config", func() {
		BeforeEach(func() {
			serverConfig.BlobsPath = ""
			marshalToFile(configFilePath, serverConfig)
		})

		It("fails with an error message", func() {
			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err).To(gbytes.Say("blobs path is required"))
		})
	})

	Context("when the config contains a cert and key", func() {
		var certs *x509.CertPool

		BeforeEach(func() {
			u.Scheme = "https"

			serverConfig.CertFile = "fixtures/certs/server.pem"
			serverConfig.KeyFile = "fixtures/certs/server.key"
			marshalToFile(configFilePath, serverConfig)

			certBytes, err := ioutil.ReadFile(serverConfig.CertFile)
			Expect(err).NotTo(HaveOccurred())

			certs = x509.NewCertPool()
			ok := certs.AppendCertsFromPEM(certBytes)
			Expect(ok).To(BeTrue())
		})

		It("enables https for the transport with the specified key and cert", func() {
			Eventually(dial("tcp", listenAddress)).Should(Succeed())

			client := http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{RootCAs: certs},
				},
			}

			resp, err := client.Get(u.String())
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})
	})
})

func marshalToFile(path string, object interface{}) {
	data, err := json.Marshal(object)
	Expect(err).NotTo(HaveOccurred())

	err = ioutil.WriteFile(path, data, 0644)
	Expect(err).NotTo(HaveOccurred())
}

func dial(network, address string) func() error {
	return func() error {
		c, err := net.Dial(network, address)
		if err != nil {
			return err
		}
		c.Close()
		return nil
	}
}
