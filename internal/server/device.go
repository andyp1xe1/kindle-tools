package server

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/skip2/go-qrcode"

	"wallpapers/internal/kindle"
)

type deviceInfo struct {
	Model     string `json:"model"`
	W         int    `json:"w"`
	H         int    `json:"h"`
	Bpp       int    `json:"bpp"`
	Grayscale bool   `json:"grayscale"`
}

func gatherDevice() deviceInfo {
	w, h, bpp, gray := kindle.ParseEips()
	if w == 0 || h == 0 {
		w, h, bpp, gray = 1072, 1448, 8, true
	}
	name := ""
	if m, ok := kindle.ModelFromSerial(); ok {
		name = m.Name
	}
	if name == "" {
		name = fmt.Sprintf("Kindle (%d×%d)", w, h)
	}
	return deviceInfo{name, w, h, bpp, gray}
}

// ---- network / URL --------------------------------------------------------

func handleQR(w http.ResponseWriter, r *http.Request) {
	log.Printf("QR fetched by UA=%q", r.UserAgent())
	url := externalURL(r)
	if url == "" {
		httpErr(w, errors.New("no usable IP"), 503)
		return
	}
	png, err := qrcode.Encode(url, qrcode.Medium, 720)
	if err != nil {
		httpErr(w, err, 500)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Write(png)
}

// handleURL returns the external-facing URL for the UI, or an empty body if
// no non-loopback IP is available (no wifi). start.sh polls this so the
// displayed URL and the QR encode the same string.
func handleURL(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintln(w, externalURL(r))
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	log.Printf("ping from %s", r.RemoteAddr)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintln(w, "wallpapers")
}

// handleKill stops the server. No token required, but localhost-only — the
// kindle's launchers (scriptlet, koplugin) and the on-device Mesquite webapp
// all reach us via 127.0.0.1; phones on the LAN can't kill the session.
func handleKill(w http.ResponseWriter, r *http.Request) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || (host != "127.0.0.1" && host != "::1") {
		log.Printf("kill rejected from %s", r.RemoteAddr)
		httpErr(w, errors.New("forbidden"), 403)
		return
	}
	log.Printf("kill from %s — shutting down", r.RemoteAddr)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(204)
	go stop()
}

func externalURL(r *http.Request) string {
	ip := kindle.DetectIP()
	if ip == "" {
		if h, _, err := net.SplitHostPort(r.Host); err == nil && h != "" && h != "127.0.0.1" && h != "localhost" {
			ip = h
		}
	}
	if ip == "" {
		return ""
	}
	return fmt.Sprintf("http://%s:%d/%s/", ip, port, token)
}
