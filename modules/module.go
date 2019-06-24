package modules

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/agile-work/srv-shared/rdb"
)

// Module defines each installed Module
type Module struct {
	Name            string `json:"name"`
	Servers         []*Server
	nextServerIndex int
}

// AvailableServers returns a list of available servers
func (m *Module) AvailableServers() []*Server {
	available := []*Server{}
	for _, s := range m.Servers {
		if s.UP {
			available = append(available, s)
		}
	}
	return available
}

// Server returns server to execute request with round robin balance
func (m *Module) Server() *Server {
	server := m.Servers[m.nextServerIndex]
	nextServerIsDown := false

	i := 0
	for server.UP == false && i < len(m.Servers) {
		nextServerIsDown = true
		server = m.Servers[i]
		i++
	}

	m.nextServerIndex++
	if nextServerIsDown {
		m.nextServerIndex = i
	}

	if m.nextServerIndex >= len(m.Servers) {
		m.nextServerIndex = 0
	}

	if server.UP == false {
		return nil
	}

	return server
}

// Match check if this service can process this path
func (m *Module) Match(path string) bool {
	pattern := fmt.Sprintf("/api/v1/%s/", strings.ToLower(m.Name))
	return strings.Contains(path, pattern)
}

// AddServer append a server to module list if it does not exist
func (m *Module) AddServer(srv *Server) {
	for _, s := range m.Servers {
		if s.InstanceCode == srv.InstanceCode || s.Addr() == srv.Addr() {
			s.UP = true
			return
		}
	}
	m.Servers = append(m.Servers, srv)
}

// Request executes the request returning a response
func (m *Module) Request(client *http.Client, path, method string, header http.Header, body io.Reader) (*http.Response, error) {
	bodyBytes, err := ioutil.ReadAll(body)
	maxRetryAttempts := len(m.Servers) - 1
	server := m.Server()
	if server == nil {
		return nil, errors.New("No available servers to answer this request 1")
	}

	// TODO: include in util shared as a function
	body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	response, err := server.Request(client, path, method, header, body)
	if err != nil {
		fmt.Println(err)
	}

	i := 0
	for (response == nil || err != nil) && i < maxRetryAttempts {
		// TODO: include in util shared as a function
		body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
		server.UP = false
		fmt.Printf("module %s(instances:%d) ready\n", m.Name, len(m.AvailableServers()))

		if err := rdb.Delete("module:def:" + server.InstanceCode); err != nil {
			return nil, err
		}
		if _, err := rdb.LRem("api:modules", 0, server.InstanceCode); err != nil {
			return nil, err
		}
		server = m.Server()
		if server == nil {
			return nil, errors.New("No available servers to answer this request 2")
		}
		response, err = server.Request(client, path, method, header, body)
		i++
	}

	if (response == nil || err != nil) && maxRetryAttempts == 0 {
		server.UP = false
		fmt.Printf("module %s(instances:%d) ready\n", m.Name, len(m.AvailableServers()))

		if err := rdb.Delete("module:def:" + server.InstanceCode); err != nil {
			return nil, err
		}
		if _, err := rdb.LRem("api:modules", 0, server.InstanceCode); err != nil {
			return nil, err
		}
		return nil, errors.New("No available servers to answer this request 3")
	}

	return response, err
}
