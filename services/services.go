package services

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/BurntSushi/toml"
)

//Services defines the structure to deal with all available services
type Services struct {
	List   []*Service `json:"list" toml:"service"`
	Client *http.Client
}

//Process request and select to correct service
func (s *Services) Process(path, method string, header http.Header, body io.Reader) (*http.Response, error) {
	for _, service := range s.List {
		if service.Match(path) {
			return service.Request(s.Client, path, method, header, body)
		}
	}
	return nil, errors.New("invalid path no service to responde")
}

//Load decode toml config file
func (s *Services) Load(verbose bool) error {
	fmt.Println("Loading services")

	if _, err := toml.DecodeFile("services.toml", s); err != nil {
		panic("Invalid services file")
	}

	for _, service := range s.List {
		service.Ping(s.Client)
		if verbose {
			fmt.Printf("%s\n", service.Name)
			for _, server := range service.Servers {
				fmt.Println(server.String())
			}
		}
	}

	return nil
}

//VerifyDownServers ping all down servers in the service
func (s *Services) VerifyDownServers() {
	for _, service := range s.List {
		service.PingDownServers(s.Client)
	}
}

var srv *Services

//New load services from file and returns a pointer
func New(certPath, keyPath string) *Services {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		panic("Invalid certificate file")
	}

	caCert, err := ioutil.ReadFile(certPath)
	if err != nil {
		panic("Invalid certificate file")
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	client := &http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:      caCertPool,
				Certificates: []tls.Certificate{cert},
			},
		},
	}

	srv = &Services{
		Client: client,
	}
	srv.Load(true)
	return srv
}
