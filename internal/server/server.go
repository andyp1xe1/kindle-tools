// Package server is the shared HTTP scaffolding for the launcher binaries.
// Server embeds an *http.ServeMux and pre-wires the scriptlet endpoints:
//
//	GET  /url    external-facing URL as text, or empty body if no IP
//	GET  /qr     PNG QR code of the external URL, or 503 if no IP
//	POST /kill   shutdown
//	GET  /ping   liveness, returns the server name
//
// /url, /qr, /kill are 127.0.0.1-only — the on-Kindle scriptlet hits them
// directly, so the token doesn't appear in scriptlet URLs. The token only
// gates the LAN-facing routes registered via HandleProtected. /ping is open
// so launcher shell scripts can probe liveness without auth.
package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/skip2/go-qrcode"

	"wallpapers/internal/kindle"
)

type Server struct {
	*http.ServeMux
	Port  int
	Token string

	name string
	dev  bool
	stop context.CancelFunc
}

// New builds a Server with a fresh token and the scriptlet endpoints wired.
// dev skips firewall opening so the binary runs on a non-Kindle host.
func New(name string, port int, dev bool) *Server {
	s := &Server{
		ServeMux: http.NewServeMux(),
		Port:     port,
		Token:    genToken(),
		name:     name,
		dev:      dev,
	}
	s.HandleFunc("GET /url", LocalhostOnly(s.handleURL))
	s.HandleFunc("GET /qr", LocalhostOnly(s.handleQR))
	s.HandleFunc("POST /kill", LocalhostOnly(s.handleKill))
	s.HandleFunc("GET /ping", s.handlePing)
	return s
}

// LocalhostOnly wraps h to reject requests not coming from 127.0.0.1 / ::1.
func LocalhostOnly(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil || (host != "127.0.0.1" && host != "::1") {
			log.Printf("%s %s rejected from %s", r.Method, r.URL.Path, r.RemoteAddr)
			http.Error(w, "forbidden", 403)
			return
		}
		h(w, r)
	}
}

// URL returns the external-facing URL for the UI, or "" if no non-loopback IP
// is available. r is optional; if set, its Host is used as a fallback when
// DetectIP fails and the caller isn't on loopback.
func (s *Server) URL(r *http.Request) string {
	ip := kindle.DetectIP()
	if ip == "" && r != nil {
		if h, _, err := net.SplitHostPort(r.Host); err == nil && h != "" && h != "127.0.0.1" && h != "localhost" {
			ip = h
		}
	}
	if ip == "" {
		return ""
	}
	return fmt.Sprintf("http://%s:%d/%s/", ip, s.Port, s.Token)
}

// HandleProtected registers h under the token prefix. Pattern is
// http.ServeMux's "[METHOD ]PATH"; the token is inserted right before PATH.
func (s *Server) HandleProtected(pattern string, h http.Handler) {
	method, path, hasMethod := strings.Cut(pattern, " ")
	if hasMethod {
		s.Handle(method+" /"+s.Token+path, h)
	} else {
		s.Handle("/"+s.Token+pattern, h)
	}
}

// HandleFuncProtected is the http.HandlerFunc variant of HandleProtected.
func (s *Server) HandleFuncProtected(pattern string, h http.HandlerFunc) {
	s.HandleProtected(pattern, h)
}

// Run serves until ctx is cancelled or /kill is hit.
func (s *Server) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	s.stop = cancel

	addr := fmt.Sprintf(":%d", s.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	if !s.dev {
		kindle.OpenFirewall(s.Port)
	}

	srv := &http.Server{Handler: s, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		log.Printf("shutdown signal received")
		shutdownCtx, cancelTimeout := context.WithTimeout(context.Background(), 5*time.Second)
		srv.Shutdown(shutdownCtx)
		cancelTimeout()
		log.Printf("shutdown complete")
	}()

	url := s.URL(nil)
	if url == "" {
		url = "(no wifi)"
	}
	log.Printf("%s startup: listening on %s  token=%s  url=%s  dev=%v", s.name, addr, s.Token, url, s.dev)

	if err := srv.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) handleURL(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	fmt.Fprintln(w, s.URL(r))
}

func (s *Server) handleQR(w http.ResponseWriter, r *http.Request) {
	log.Printf("QR fetched by UA=%q", r.UserAgent())
	url := s.URL(r)
	if url == "" {
		http.Error(w, "no usable IP", 503)
		return
	}
	png, err := qrcode.Encode(url, qrcode.Medium, 720)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Write(png)
}

func (s *Server) handlePing(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintln(w, s.name)
}

func (s *Server) handleKill(w http.ResponseWriter, r *http.Request) {
	log.Printf("kill from %s — shutting down", r.RemoteAddr)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(204)
	go s.stop()
}

// genToken returns 8 hex chars (~32 bits entropy).
func genToken() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("rand: %v", err)
	}
	return hex.EncodeToString(b)
}
