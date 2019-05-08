package services

import (
	"fmt"
	"io"
	"net/http"
)

//Server defines the connection to this service in a server
type Server struct {
	Name string `json:"name"`
	Host string `json:"host"`
	Port int    `json:"port"`
	UP   bool   `json:"up"`
}

//Request executes the request to a server returning a response
func (s *Server) Request(client *http.Client, path, method string, header http.Header, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, s.URL(path), body)
	if err != nil {
		return nil, err
	}
	req.Header = header
	return client.Do(req)
}

//Ping check server is alive
func (s *Server) Ping(client *http.Client) bool {
	resp, _ := s.Request(client, "/ping", http.MethodGet, nil, nil)
	if resp != nil && resp.StatusCode == http.StatusOK {
		return true
	}
	return false
}

//URL returns server URL to some path
func (s *Server) URL(path string) string {
	return fmt.Sprintf("https://%s:%d%s", s.Host, s.Port, path)
}

//URL returns server informations
func (s *Server) String() string {
	return fmt.Sprintf("Server: %s (%s:%d) - UP:%t", s.Name, s.Host, s.Port, s.UP)
}
