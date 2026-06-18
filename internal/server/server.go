// Package server runs the wallpaper manager HTTP server.
//
// Layout under /usr/share/blanket:
//
//	wallpapers/<preset>/bg_ssNN.png  — user-owned preset directories
//	screensaver -> wallpapers/<preset>  — symlink the OS reads from
//
// "Activating" a preset swaps the symlink atomically; uploads land inside the
// preset dir directly, so they show up on the next screen-off without any
// extra plumbing.
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
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"syscall"
	"time"

	"wallpapers/internal/kindle"
	"wallpapers/internal/web"
)

const (
	blanketDir = "/usr/share/blanket"
	devDir     = "./dev"
	tokenFile  = "/tmp/wallpapers.token"
)

// Config is the runtime configuration passed by main.
type Config struct {
	Port    int
	DevMode bool
}

var (
	port    int
	devMode bool
	blanket string
	token   string // random per-startup URL prefix; gates the whole UI + API
	stop    context.CancelFunc

	ssNameRE     = regexp.MustCompile(`^bg_ss\d{2,3}\.png$`)
	presetNameRE = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}$`)
)

// routes registers all handlers
func routes(mux *http.ServeMux, pfx string) {
	mux.Handle(pfx+"/", http.StripPrefix(pfx, http.FileServer(http.FS(web.FS))))
	mux.HandleFunc("GET "+pfx+"/api/state", handleState)
	mux.HandleFunc("POST "+pfx+"/api/claim", handleClaim)
	mux.HandleFunc("POST "+pfx+"/api/presets", handleCreatePreset)
	mux.HandleFunc("DELETE "+pfx+"/api/presets/{name}", handleDeletePreset)
	mux.HandleFunc("POST "+pfx+"/api/presets/{name}/activate", handleActivatePreset)
	mux.HandleFunc("POST "+pfx+"/api/presets/{name}/rename", handleRenamePreset)
	mux.HandleFunc("GET "+pfx+"/api/presets/{name}/files", handleListFiles)
	mux.HandleFunc("GET "+pfx+"/api/presets/{name}/files/{file}", handleGetFile)
	mux.HandleFunc("POST "+pfx+"/api/presets/{name}/files", handleUploadFile)
	mux.HandleFunc("DELETE "+pfx+"/api/presets/{name}/files/{file}", handleDeleteFile)
	mux.HandleFunc("POST "+pfx+"/api/import", handleImport)
	mux.HandleFunc("GET "+pfx+"/qr", handleQR)
	mux.HandleFunc("GET "+pfx+"/url", handleURL)
	mux.HandleFunc("GET /ping", handlePing)
	mux.HandleFunc("POST /kill", handleKill)
}

// Run binds, writes the readiness token file, and serves until the listener
// closes. It returns the underlying serve error.
func Run(c Config) error {
	port = c.Port
	devMode = c.DevMode
	if devMode {
		blanket = devDir
	} else {
		blanket = blanketDir
	}
	if err := withRW(initLayout); err != nil {
		return fmt.Errorf("init: %w", err)
	}
	token = genToken()

	mux := http.NewServeMux()
	routes(mux, "/"+token)

	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	if !devMode {
		kindle.OpenFirewall(port)
	}
	if err := os.WriteFile(tokenFile, []byte(token), 0600); err != nil {
		log.Printf("warning: could not write %s: %v", tokenFile, err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	stop = cancel
	defer os.Remove(tokenFile)

	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		log.Printf("shutdown signal received")
		shutdownCtx, cancelTimeout := context.WithTimeout(context.Background(), 5*time.Second)
		srv.Shutdown(shutdownCtx)
		cancelTimeout()
		log.Printf("shutdown complete")
	}()

	ip := kindle.DetectIP()
	url := "(no wifi)"
	if ip != "" {
		url = fmt.Sprintf("http://%s:%d/%s/", ip, port, token)
	}
	log.Printf("wallpapers startup: listening on %s  token=%s  url=%s  blanket=%s  dev=%v", addr, token, url, blanket, devMode)

	if err := srv.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// genToken returns 8 hex chars (~32 bits entropy)
func genToken() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("rand: %v", err)
	}
	return hex.EncodeToString(b)
}

func initLayout() error {
	return os.MkdirAll(filepath.Join(blanket, "wallpapers"), 0755)
}
