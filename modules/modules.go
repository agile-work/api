package modules

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/agile-work/srv-shared/rdb"
)

// Modules defines the structure to deal with all available modules
type Modules struct {
	List   map[string]*Module
	Client *http.Client
}

// Process request and select to correct service
func (mdls *Modules) Process(path, method string, header http.Header, body io.Reader) (*http.Response, error) {
	for _, module := range mdls.List {
		if module.Match(path) {
			return module.Request(mdls.Client, path, method, header, body)
		}
	}
	return nil, errors.New("no service to responde")
}

// Load read modules list from Redis
func (mdls *Modules) Load() error {
	fmt.Println("Loading modules...")

	modulesInstanceIDs, err := rdb.LRange("api:modules", 0, -1)
	if err != nil {
		return err
	}

	for _, instanceCode := range modulesInstanceIDs {
		moduleDefRedisKey := fmt.Sprintf("module:def:%s", instanceCode)
		moduleDefJSON, err := rdb.Get(moduleDefRedisKey)
		if err != nil {
			return err
		}

		srv := &Server{}
		err = json.Unmarshal([]byte(moduleDefJSON), srv)
		if err != nil {
			return err
		}
		srv.UP = srv.Ping(mdls.Client)
		if !srv.UP {
			if err := rdb.Delete(moduleDefRedisKey); err != nil {
				return err
			}
			if _, err := rdb.LRem("api:modules", 0, moduleDefRedisKey); err != nil {
				return err
			}
			continue
		}

		if mdl, ok := mdls.List[srv.Name]; ok {
			mdl.AddServer(srv)
		} else {
			mdl := &Module{
				Name: srv.Name,
			}
			mdl.Servers = append(mdl.Servers, srv)
			mdls.List[srv.Name] = mdl
		}
	}

	if len(modulesInstanceIDs) == 0 {
		mdls.List = make(map[string]*Module)
	}

	for _, m := range mdls.List {
		fmt.Printf("module %s(instances:%d) ready\n", m.Name, len(m.Servers))
	}

	if len(mdls.List) == 0 {
		fmt.Printf("no modules loaded\n")
	}

	return nil
}

// New load services from file and returns a pointer
func New(certPath, keyPath string) *Modules {
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

	mdls := &Modules{
		Client: client,
		List:   make(map[string]*Module),
	}

	if err := mdls.Load(); err != nil {
		fmt.Println(err)
	}

	return mdls
}
