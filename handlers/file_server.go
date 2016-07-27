package handlers

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const REDIRECT_SUFFIX = ".redirect"

type FileServer struct {
	Root string
}

func (fs *FileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	upath := r.URL.Path
	if !strings.HasPrefix(upath, "/") {
		upath = "/" + upath
		r.URL.Path = upath
	}

	upath = path.Clean(upath)
	if strings.Contains(upath, "..") || strings.Contains(upath, "\x00") {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	location := filepath.Join(fs.Root, upath)
	log.Printf("method: %s, location: %s", r.Method, location)

	switch r.Method {
	case http.MethodGet, http.MethodHead:
		redirect, err := ioutil.ReadFile(location + REDIRECT_SUFFIX)
		if err == nil {
			http.Redirect(w, r, string(redirect), http.StatusTemporaryRedirect)
			return
		}
		httpFS := http.FileServer(http.Dir(fs.Root))
		httpFS.ServeHTTP(w, r)

	case http.MethodPut:
		err := os.MkdirAll(filepath.Dir(location), 0755)
		if err != nil {
			sendErrorResponse(w, r, err)
			return
		}

		output, err := os.OpenFile(location, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
		if err != nil {
			sendErrorResponse(w, r, err)
			return
		}
		defer output.Close()

		_, err = io.Copy(output, r.Body)
		if err != nil {
			sendErrorResponse(w, r, err)
			return
		}

		w.WriteHeader(http.StatusCreated)

	case http.MethodDelete:
		err := os.Remove(location)
		if err != nil {
			sendErrorResponse(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}

func sendErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case os.IsExist(err):
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			w.WriteHeader(http.StatusConflict)
		}
	case os.IsNotExist(err):
		w.WriteHeader(http.StatusNotFound)
	case os.IsPermission(err):
		w.WriteHeader(http.StatusForbidden)
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
	return
}
