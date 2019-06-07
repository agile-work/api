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

	"github.com/agile-work/api/middlewares"
	"github.com/agile-work/srv-shared/sql-builder/db"

	"github.com/agile-work/api/services"
	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
)

var (
	addr     = flag.String("port", ":8080", "TCP port to listen to")
	cert     = flag.String("cert", "cert.pem", "Path to certification")
	key      = flag.String("key", "key.pem", "Path to certification key")
	host     = "cryo.cdnm8viilrat.us-east-2.rds-preview.amazonaws.com"
	port     = 5432
	user     = "cryoadmin"
	password = "x3FhcrWDxnxCq9p"
	dbName   = "cryo"
)

func main() {
	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt)

	flag.Parse()
	s := services.New(*cert, *key)

	err := db.Connect(host, port, user, password, dbName, false)
	if err != nil {
		fmt.Println("Error connecting to database")
		return
	}
	fmt.Println("Database connected")

	r := chi.NewRouter()

	cors := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Language", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	})

	r.Use(
		cors.Handler,
		middlewares.Authorization,
		middlewares.GetUser,
	)

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
