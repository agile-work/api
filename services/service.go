package services

import (
	"errors"
	"io"
	"net/http"
	"strings"
)

//Service defines each installed service
type Service struct {
	Name            string    `json:"name"`
	Type            string    `json:"type"`
	Servers         []*Server `json:"servers" toml:"server"`
	NextServerIndex int       `json:"nextServerIndex"`
}

//Server returns server to execute request with round robin balance
func (s *Service) Server() *Server {
	server := s.Servers[s.NextServerIndex]
	NextServerIsDown := false

	i := 0
	for server.UP == false && i < len(s.Servers) {
		NextServerIsDown = true
		server = s.Servers[i]
		i++
	}

	s.NextServerIndex++
	if NextServerIsDown {
		s.NextServerIndex = i
	}

	if s.NextServerIndex >= len(s.Servers) {
		s.NextServerIndex = 0
	}

	if server.UP == false {
		return nil
	}

	return server
}

//Match check if this service can process this path
func (s *Service) Match(path string) bool {
	return strings.Contains(path, s.Type)
}

//Request executes the request returning a response
func (s *Service) Request(client *http.Client, path, method string, header http.Header, body io.Reader) (*http.Response, error) {
	maxRetryAttempts := len(s.Servers) - 1
	server := s.Server()
	if server == nil {
		return nil, errors.New("No available servers to answer this request")
	}
	response, err := server.Request(client, path, method, header, body)

	i := 0
	for response == nil && i < maxRetryAttempts {
		server.UP = false
		server = s.Server()
		response, err = server.Request(client, path, method, header, body)
		i++
	}
	return response, err
}

//Ping check service is alive
func (s *Service) Ping(client *http.Client) {
	for i, server := range s.Servers {
		s.Servers[i].UP = server.Ping(client)
	}
}

//PingDownServers ping down servers from this service
func (s *Service) PingDownServers(client *http.Client) {
	for i, server := range s.Servers {
		if server.UP == false {
			s.Servers[i].UP = server.Ping(client)
		}
	}
}
