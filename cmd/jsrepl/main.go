// Command jsrepl runs the reverse JS REPL as a standalone HTTP server.
// The wallpapers Mesquite widget loads /executor.js to install a silent
// long-poll executor that runs jobs sent from the laptop UI at /.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"wallpapers/internal/jsrepl"
	"wallpapers/internal/kindle"
)

func main() {
	port := flag.Int("port", 6970, "http port")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mux := http.NewServeMux()
	jsrepl.Register(mux, "")
	mux.HandleFunc("GET /ip", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		fmt.Fprintln(w, kindle.DetectIP())
	})
	mux.HandleFunc("POST /kill", func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil || (host != "127.0.0.1" && host != "::1") {
			log.Printf("kill rejected from %s", r.RemoteAddr)
			http.Error(w, "forbidden", 403)
			return
		}
		log.Printf("kill from %s — shutting down", r.RemoteAddr)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(204)
		go func() {
			time.Sleep(200 * time.Millisecond)
			cancel()
		}()
	})

	addr := fmt.Sprintf(":%d", *port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	kindle.OpenFirewall(*port)

	log.Printf("jsrepl startup: listening on %s  host=http://<ip>:%d/", addr, *port)

	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		log.Printf("shutdown signal received")
		shutdownCtx, c := context.WithTimeout(context.Background(), 3*time.Second)
		defer c()
		srv.Shutdown(shutdownCtx)
		log.Printf("shutdown complete")
	}()
	if err := srv.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
