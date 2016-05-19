package handlers_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/sykesm/dav-blobstore/handlers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Authentication", func() {
	var (
		handler  *handlers.AuthenticationHandler
		response *httptest.ResponseRecorder
	)

	BeforeEach(func() {
		response = httptest.NewRecorder()

		delegate := func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusOK)
		}

		handler = &handlers.AuthenticationHandler{
			PublicRead: false,
			Delegate:   http.HandlerFunc(delegate),
		}
	})

	It("disallows unauthenticated requests", func() {
		for _, method := range []string{"GET", "HEAD", "PUT", "POST", "DELETE", "MKCOL", "UNNOWN"} {
			req, err := http.NewRequest(method, "http://example.com/", nil)
			Expect(err).NotTo(HaveOccurred())

			handler.ServeHTTP(response, req)
			Expect(response.Code).To(Equal(http.StatusUnauthorized))
		}
	})

	Context("when PublicRead is true", func() {
		BeforeEach(func() {
			handler.PublicRead = true
		})

		It("allows unauthenticated GET requests", func() {
			req, err := http.NewRequest(http.MethodGet, "http://example.com/", nil)
			Expect(err).NotTo(HaveOccurred())

			handler.ServeHTTP(response, req)
			Expect(response.Code).To(Equal(http.StatusOK))
		})

		It("allows unauthenticated HEAD requests", func() {
			req, err := http.NewRequest(http.MethodHead, "http://example.com/", nil)
			Expect(err).NotTo(HaveOccurred())

			handler.ServeHTTP(response, req)
			Expect(response.Code).To(Equal(http.StatusOK))
		})

		It("disallows other requests", func() {
			for _, method := range []string{"PUT", "POST", "DELETE", "MKCOL", "UNNOWN"} {
				req, err := http.NewRequest(method, "http://example.com/", nil)
				Expect(err).NotTo(HaveOccurred())

				handler.ServeHTTP(response, req)
				Expect(response.Code).To(Equal(http.StatusUnauthorized))
			}
		})
	})

	Describe("requests with authentication", func() {
		BeforeEach(func() {
			handler.Authorized = map[string]string{
				"user": "password",
			}
		})

		Context("when the user and password are not in the authorized list", func() {
			It("rejects the request", func() {
				for _, method := range []string{"GET", "HEAD", "PUT", "POST", "DELETE", "MKCOL", "UNNOWN"} {
					req, err := http.NewRequest(method, "http://example.com/", nil)
					Expect(err).NotTo(HaveOccurred())

					req.SetBasicAuth("user", "bad-password")
					handler.ServeHTTP(response, req)
					Expect(response.Code).To(Equal(http.StatusForbidden))
				}
			})
		})

		Context("when the user and password are in the authorized list", func() {
			It("accepts the request", func() {
				for _, method := range []string{"GET", "HEAD", "PUT", "POST", "DELETE", "MKCOL", "UNNOWN"} {
					req, err := http.NewRequest(method, "http://example.com/", nil)
					Expect(err).NotTo(HaveOccurred())

					req.SetBasicAuth("user", "password")
					handler.ServeHTTP(response, req)
					Expect(response.Code).To(Equal(http.StatusOK))
				}
			})
		})
	})
})
