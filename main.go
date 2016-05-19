package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/sykesm/dav-blobstore/handlers"
)

type Config struct {
	BlobsPath  string            `json:"blobs_path"`
	PublicRead bool              `json:"public_read"`
	CertFile   string            `json:"cert_file,omitempty"`
	KeyFile    string            `json:"key_file,omitempty"`
	Users      map[string]string `json:"users"`
}

var configFile = flag.String(
	"configFile",
	"config.json",
	"The path to the configuration file",
)

var listenAddress = flag.String(
	"listenAddress",
	"0.0.0.0:14000",
	"The host:port address to bind to",
)

func main() {
	flag.Parse()

	config, err := loadConfig(*configFile)
	if err != nil {
		log.Fatalf("failed to load config data: %s", err)
	}

	if config.BlobsPath == "" {
		log.Fatal("blobs path is required")
	}

	fileServer := &handlers.AuthenticationHandler{
		PublicRead: config.PublicRead,
		Authorized: config.Users,
		Delegate: &handlers.FileServer{
			Root: config.BlobsPath,
		},
	}

	if config.CertFile != "" && config.KeyFile != "" {
		err = http.ListenAndServeTLS(*listenAddress, config.CertFile, config.KeyFile, fileServer)
	} else {
		err = http.ListenAndServe(*listenAddress, fileServer)
	}
	if err != nil {
		log.Fatalf("listen and serve failed: %s", err)
	}
}

func loadConfig(configFile string) (*Config, error) {
	reader, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	config := Config{}
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
