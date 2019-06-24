package modules

import (
	"fmt"
	"io"
	"net/http"
)

// Server defines the connection to this service in a server
type Server struct {
	InstanceCode string `json:"instance_code"`
	Name         string `json:"name"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
	PID          int    `json:"pid"`
	UP           bool
}

// Request executes the request to a server returning a response
func (s *Server) Request(client *http.Client, path, method string, header http.Header, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, s.URL(path), body)
	if err != nil {
		return nil, err
	}
	req.Header = header
	return client.Do(req)
}

// Addr returns host:port from theis server
func (s *Server) Addr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// URL returns server URL to some path
func (s *Server) URL(path string) string {
	return fmt.Sprintf("https://%s:%d%s", s.Host, s.Port, path)
}
