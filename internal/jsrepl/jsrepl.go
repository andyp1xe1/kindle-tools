// Package jsrepl exposes a reverse JS REPL: the Kindle's browser long-polls
// for snippets, evals them, and posts results back. A laptop UI on the same
// server lets you type into a textarea and watch results stream in.
package jsrepl

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"sync/atomic"
	"time"

	"wallpapers/internal/server"
)

//go:embed executor.js repl.html
var assets embed.FS

type Job struct {
	ID   int       `json:"id"`
	At   time.Time `json:"at"`
	Code string    `json:"code"`
}

type Result struct {
	ID    int       `json:"id"`
	At    time.Time `json:"at"`
	OK    bool      `json:"ok"`
	Value string    `json:"value,omitempty"`
	Error string    `json:"error,omitempty"`
}

var (
	inbox     = make(chan Job, maxQueue)
	outbox    = make(chan Result, maxResults)
	nextJobID atomic.Int64
)

// Register mounts the REPL endpoints: Kindle-side (executor + long-poll)
// unprefixed and localhost-only, laptop-side (UI + cmd/outbox) under /{TOK}.
func Register(s *server.Server) {
	sub, err := fs.Sub(assets, ".")
	if err != nil {
		panic(err)
	}
	s.HandleFunc("GET /executor.js", server.LocalhostOnly(serveAsset(sub, "executor.js", "application/javascript; charset=utf-8")))
	s.HandleFunc("GET /replin", server.LocalhostOnly(handleReplIn))
	s.HandleFunc("POST /replout", server.LocalhostOnly(handleReplOut))

	s.HandleFuncProtected("GET /{$}", serveAsset(sub, "repl.html", "text/html; charset=utf-8"))
	s.HandleFuncProtected("POST /inbox", handleInbox)
	s.HandleFuncProtected("GET /outbox", handleOutbox)
}

func serveAsset(sub fs.FS, name, contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := fs.ReadFile(sub, name)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "no-store")
		w.Write(b)
	}
}

// handleReplIn: Kindle long-polls the inbox for the next snippet.
func handleReplIn(w http.ResponseWriter, r *http.Request) {
	select {
	case j := <-inbox:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(j)
	case <-time.After(longPollWait):
		w.WriteHeader(http.StatusNoContent)
	case <-r.Context().Done():
	}
}

// handleReplOut: Kindle posts a result into the outbox.
func handleReplOut(w http.ResponseWriter, r *http.Request) {
	var res Result
	if err := json.NewDecoder(r.Body).Decode(&res); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	res.At = time.Now()
	outbox <- res
	w.WriteHeader(http.StatusNoContent)
}

// handleInbox: laptop pushes a snippet into the inbox.
func handleInbox(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if body.Code == "" {
		http.Error(w, "empty code", 400)
		return
	}
	j := Job{ID: int(nextJobID.Add(1)), At: time.Now(), Code: body.Code}
	inbox <- j
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(j)
}

// handleOutbox: laptop long-polls for results, draining whatever's buffered.
func handleOutbox(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var results []Result
	select {
	case res := <-outbox:
		results = append(results, res)
	case <-time.After(longPollWait):
		w.Write([]byte("[]"))
		return
	case <-r.Context().Done():
		return
	}
	for {
		select {
		case res := <-outbox:
			results = append(results, res)
		default:
			json.NewEncoder(w).Encode(results)
			return
		}
	}
}
