package main

import (
	"context"
	"flag"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/agile-work/api/services"
	"github.com/go-chi/chi"
)

var (
	addr = flag.String("port", ":8080", "TCP port to listen to")
	cert = flag.String("cert", "cert.pem", "Path to certification")
	key  = flag.String("key", "key.pem", "Path to certification key")
)

func main() {
	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt)

	flag.Parse()
	s := services.New(*cert, *key)

	r := chi.NewRouter()

	r.Route("/api/v1", func(r chi.Router) {
		r.Mount("/admin/services", services.Routes())
		r.HandleFunc("/*", func(rw http.ResponseWriter, r *http.Request) {
			path := html.EscapeString(r.URL.Path)
			resp, err := s.Process(path, r.Method, r.Header, r.Body)
			if err != nil {
				rw.WriteHeader(http.StatusNotFound)
				rw.Write([]byte(err.Error()))
				return
			}
			defer resp.Body.Close()
			for name, values := range resp.Header {
				rw.Header()[name] = values
			}
			rw.WriteHeader(resp.StatusCode)
			io.Copy(rw, resp.Body)
		})
	})

	srv := &http.Server{
		Addr:         *addr,
		Handler:      r,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		fmt.Println("")
		fmt.Println("API listening on ", *addr)
		fmt.Println("")
		if err := srv.ListenAndServeTLS(*cert, *key); err != nil {
			fmt.Printf("listen: %s\n", err)
		}
	}()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	go func() {
		for {
			select {
			case <-ticker.C:
				s.VerifyDownServers()
			}
		}
	}()

	<-stopChan
	fmt.Println("Shutting down API...")
	ticker.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	srv.Shutdown(ctx)
	defer cancel()
	fmt.Println("API stopped!")
}
