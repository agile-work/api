package main

import (
	"context"
	"flag"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/agile-work/api/middlewares"
	"github.com/agile-work/srv-shared/constants"
	"github.com/agile-work/srv-shared/rdb"
	"github.com/agile-work/srv-shared/service"
	"github.com/agile-work/srv-shared/socket"

	"github.com/agile-work/api/modules"
	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
)

var (
	cert      = flag.String("cert", "cert.pem", "Path to certification")
	key       = flag.String("key", "key.pem", "Path to certification key")
	host      = flag.String("host", "localhost", "TCP hostname to connect to")
	port      = flag.Int("port", 8080, "TCP port to listen to")
	redisHost = flag.String("redisHost", "localhost", "Redis host")
	redisPort = flag.Int("redisPort", 6379, "Redis port")
	redisPass = flag.String("redisPass", "redis123", "Redis password")
	wsHost    = flag.String("wsHost", "localhost", "Realtime host")
	wsPort    = flag.Int("wsPort", 8010, "Realtime port")
)

func main() {
	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt)

	flag.Parse()

	pid := os.Getpid()
	api := service.New("API", constants.ServiceTypeAPI, *host, *port, pid)

	fmt.Println("Starting API...")
	fmt.Printf("[Instance: %s | PID: %d]\n", api.InstanceCode, api.PID)

	socket.Init(api, *wsHost, *wsPort)
	defer socket.Close()

	rdb.Init(*redisHost, *redisPort, *redisPass)
	defer rdb.Close()

	var mdls *modules.Modules

	deadline := time.Now().Add(15 * time.Second)
	for {
		if rdb.Available() || time.Now().After(deadline) {
			mdls = modules.New(*cert, *key)
			break
		}
	}

	r := chi.NewRouter()

	cors := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Language", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	})

	r.Use(
		cors.Handler,
		middlewares.Authorization,
	)

	r.Route("/api/v1", func(r chi.Router) {
		r.HandleFunc("/*", func(rw http.ResponseWriter, r *http.Request) {
			path := html.EscapeString(r.URL.RequestURI())
			resp, err := mdls.Process(path, r.Method, r.Header, r.Body)
			if err != nil {
				rw.WriteHeader(http.StatusServiceUnavailable)
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
		Addr:         api.URL(),
		Handler:      r,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		fmt.Printf("API ready listen on %d\n", api.Port)
		if err := srv.ListenAndServeTLS(*cert, *key); err != nil {
			if strings.Contains(err.Error(), "bind: address already in use") {
				fmt.Println("")
				log.Fatalf("port %d already in use\n", api.Port)
			}
			fmt.Printf("listen: %s\n", err)
		}
	}()

	realtimeMessages := socket.MessagesChannel()
	go func() {
		for msg := range realtimeMessages {
			msgStr := msg.Data.(string)
			if msgStr == "reload" {
				if err := mdls.Load(); err != nil {
					fmt.Println(err)
				}
			}
		}
	}()

	<-stopChan
	fmt.Println("\nShutting down API...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	srv.Shutdown(ctx)
	defer cancel()
	fmt.Println("API stopped!")
}
