package handlers_test

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/sykesm/dav-blobstore/handlers"
)

var _ = Describe("FileServer", func() {
	var (
		handler  *handlers.FileServer
		response *httptest.ResponseRecorder

		tempDir string
	)

	BeforeEach(func() {
		var err error
		tempDir, err = ioutil.TempDir("", "file-server")
		Expect(err).NotTo(HaveOccurred())

		response = httptest.NewRecorder()
		handler = &handlers.FileServer{
			Root: tempDir,
		}

		log.SetOutput(GinkgoWriter)
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	Describe("GET", func() {
		var file string

		BeforeEach(func() {
			file = filepath.Join(tempDir, "file.txt")
			err := ioutil.WriteFile(file, []byte("blob-data"), 0644)
			Expect(err).NotTo(HaveOccurred())
		})

		It("retrieves the file contents", func() {
			req, err := http.NewRequest(http.MethodGet, "http://example.com/file.txt", nil)
			Expect(err).NotTo(HaveOccurred())

			handler.ServeHTTP(response, req)

			Expect(response.Code).To(Equal(http.StatusOK))
			Expect(response.Body.Bytes()).To(BeEquivalentTo("blob-data"))
		})
	})

	Describe("HEAD", func() {
		var file string

		BeforeEach(func() {
			file = filepath.Join(tempDir, "file.txt")
			err := ioutil.WriteFile(file, []byte("blob-data"), 0644)
			Expect(err).NotTo(HaveOccurred())
		})

		It("retrieves the file attributes", func() {
			req, err := http.NewRequest(http.MethodHead, "http://example.com/file.txt", nil)
			Expect(err).NotTo(HaveOccurred())

			handler.ServeHTTP(response, req)

			Expect(response.Code).To(Equal(http.StatusOK))
			Expect(response.HeaderMap).To(HaveKey("Last-Modified"))
			Expect(response.HeaderMap).To(HaveKeyWithValue("Content-Length", []string{"9"}))
		})
	})

	Describe("PUT", func() {
		It("creates the target file", func() {
			req, err := http.NewRequest(http.MethodPut, "http://example.com/file.txt", nil)
			Expect(err).NotTo(HaveOccurred())

			req.Body = ioutil.NopCloser(strings.NewReader("blob-data"))
			handler.ServeHTTP(response, req)

			Expect(response.Code).To(Equal(http.StatusCreated))

			file := filepath.Join(tempDir, "file.txt")
			Expect(file).To(BeARegularFile())

			contents, err := ioutil.ReadFile(file)
			Expect(err).NotTo(HaveOccurred())
			Expect(contents).To(BeEquivalentTo("blob-data"))
		})

		Context("when the target path contains directories", func() {
			It("generates intermediate directories", func() {
				req, err := http.NewRequest(http.MethodPut, "http://example.com/subdir1/subdir2/file.txt", nil)
				Expect(err).NotTo(HaveOccurred())

				req.Body = ioutil.NopCloser(strings.NewReader("blob-data"))
				handler.ServeHTTP(response, req)

				Expect(response.Code).To(Equal(http.StatusCreated))

				Expect(filepath.Join(tempDir, "subdir1")).To(BeADirectory())
				Expect(filepath.Join(tempDir, "subdir1", "subdir2")).To(BeADirectory())

				file := filepath.Join(tempDir, "subdir1", "subdir2", "file.txt")
				Expect(file).To(BeARegularFile())

				contents, err := ioutil.ReadFile(file)
				Expect(err).NotTo(HaveOccurred())
				Expect(contents).To(BeEquivalentTo("blob-data"))
			})
		})

		Context("when the file already exists", func() {
			BeforeEach(func() {
				ioutil.WriteFile(filepath.Join(tempDir, "file.txt"), []byte("blob-data"), 0644)
			})

			It("fails with 409 Conflict", func() {
				req, err := http.NewRequest(http.MethodPut, "http://example.com/file.txt", nil)
				Expect(err).NotTo(HaveOccurred())

				req.Body = ioutil.NopCloser(strings.NewReader("blob-data"))
				handler.ServeHTTP(response, req)

				Expect(response.Code).To(Equal(http.StatusConflict))
			})
		})

		Context("when the target directory cannot be written to", func() {
			BeforeEach(func() {
				os.Mkdir(filepath.Join(tempDir, "subdir"), 0550)
			})

			It("fails with 403 Forbidden", func() {
				req, err := http.NewRequest(http.MethodPut, "http://example.com/subdir/file.txt", nil)
				Expect(err).NotTo(HaveOccurred())

				req.Body = ioutil.NopCloser(strings.NewReader("blob-data"))
				handler.ServeHTTP(response, req)

				Expect(response.Code).To(Equal(http.StatusForbidden))
			})
		})
	})

	Describe("DELETE", func() {
		var dir, file string

		BeforeEach(func() {
			dir = filepath.Join(tempDir, "subdir")
			err := os.Mkdir(filepath.Join(tempDir, "subdir"), 0755)
			Expect(err).NotTo(HaveOccurred())

			file = filepath.Join(dir, "file.txt")
			err = ioutil.WriteFile(file, []byte("blob-data"), 0644)
			Expect(err).NotTo(HaveOccurred())
		})

		It("deletes the file", func() {
			req, err := http.NewRequest(http.MethodDelete, "http://example.com/subdir/file.txt", nil)
			Expect(err).NotTo(HaveOccurred())

			handler.ServeHTTP(response, req)

			Expect(response.Code).To(Equal(http.StatusNoContent))
			Expect(file).NotTo(BeAnExistingFile())
		})

		It("does not delete the directory", func() {
			req, err := http.NewRequest(http.MethodDelete, "http://example.com/subdir/file.txt", nil)
			Expect(err).NotTo(HaveOccurred())

			handler.ServeHTTP(response, req)

			Expect(response.Code).To(Equal(http.StatusNoContent))
			Expect(dir).To(BeADirectory())
		})

		Context("when the target is an empty directory", func() {
			var emptyDir string

			BeforeEach(func() {
				emptyDir := filepath.Join(tempDir, "empty-subdir")
				err := os.Mkdir(emptyDir, 0755)
				Expect(err).NotTo(HaveOccurred())
			})

			It("deletes the directory", func() {
				req, err := http.NewRequest(http.MethodDelete, "http://example.com/empty-subdir", nil)
				Expect(err).NotTo(HaveOccurred())

				handler.ServeHTTP(response, req)

				Expect(response.Code).To(Equal(http.StatusNoContent))
				Expect(emptyDir).NotTo(BeADirectory())
			})
		})

		Context("when the target is a directory with content", func() {
			It("fails with a bad request", func() {
				req, err := http.NewRequest(http.MethodDelete, "http://example.com/subdir", nil)
				Expect(err).NotTo(HaveOccurred())

				handler.ServeHTTP(response, req)

				Expect(response.Code).To(Equal(http.StatusBadRequest))
			})
		})

		Context("when the file doesn't exist", func() {
			It("fails with 404 NotFound", func() {
				req, err := http.NewRequest(http.MethodDelete, "http://example.com/subdir/missing.txt", nil)
				Expect(err).NotTo(HaveOccurred())

				handler.ServeHTTP(response, req)

				Expect(response.Code).To(Equal(http.StatusNotFound))
			})
		})

		Context("when the can't be deteled due to permissions", func() {
			BeforeEach(func() {
				err := os.Chmod(dir, 0550)
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				err := os.Chmod(file, 0755)
				Expect(err).NotTo(HaveOccurred())
			})

			It("fails with 403 Forbidden", func() {
				req, err := http.NewRequest(http.MethodDelete, "http://example.com/subdir/file.txt", nil)
				Expect(err).NotTo(HaveOccurred())

				handler.ServeHTTP(response, req)

				Expect(response.Code).To(Equal(http.StatusForbidden))
			})
		})
	})

	Context("when the cleaned path contains ..", func() {
		It("rejects the request", func() {
			req, err := http.NewRequest("anything", "http://example.com/%2e%2efoo", nil)
			Expect(err).NotTo(HaveOccurred())

			handler.ServeHTTP(response, req)

			Expect(response.Code).To(Equal(http.StatusBadRequest))
		})
	})

	Context("when the cleaned path contains \x00", func() {
		It("rejects the request", func() {
			req, err := http.NewRequest("anything", "http://example.com/foo\x00bar", nil)
			Expect(err).NotTo(HaveOccurred())

			handler.ServeHTTP(response, req)

			Expect(response.Code).To(Equal(http.StatusBadRequest))
		})
	})
})
