// Package jsrepl exposes a reverse JS REPL: the Kindle's browser long-polls
// for snippets, evals them, and posts results back. A laptop UI on the same
// server lets you type into a textarea and watch results stream in.
//
// Self-contained — remove this directory and the Register call in
// internal/server/server.go to drop the feature entirely.
package jsrepl

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

//go:embed executor.js repl.html
var assets embed.FS

const (
	longPollWait = 25 * time.Second
	maxQueue     = 100
	maxResults   = 200
)

type Job struct {
	ID   int    `json:"id"`
	Code string `json:"code"`
}

type Result struct {
	ID    int    `json:"id"`
	OK    bool   `json:"ok"`
	Value string `json:"value,omitempty"`
	Error string `json:"error,omitempty"`
	At    int64  `json:"at"`
}

type hub struct {
	mu       sync.Mutex
	nextID   int
	queue    []Job
	queueSig chan struct{}
	results  []Result
	resSig   chan struct{}
}

var h = &hub{
	queueSig: make(chan struct{}),
	resSig:   make(chan struct{}),
}

// tryDequeue pops a job if available, otherwise returns the signal channel
// to wait on. Taking the lock once avoids the lost-wakeup race between
// checking the queue and subscribing for new items.
func (h *hub) tryDequeue() (Job, bool, <-chan struct{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.queue) > 0 {
		j := h.queue[0]
		h.queue = h.queue[1:]
		return j, true, nil
	}
	return Job{}, false, h.queueSig
}

func (h *hub) enqueue(code string) Job {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.nextID++
	j := Job{ID: h.nextID, Code: code}
	h.queue = append(h.queue, j)
	if len(h.queue) > maxQueue {
		h.queue = h.queue[len(h.queue)-maxQueue:]
	}
	close(h.queueSig)
	h.queueSig = make(chan struct{})
	return j
}

func (h *hub) addResult(r Result) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.results = append(h.results, r)
	if len(h.results) > maxResults {
		h.results = h.results[len(h.results)-maxResults:]
	}
	close(h.resSig)
	h.resSig = make(chan struct{})
}

func (h *hub) resultsSince(since int) ([]Result, <-chan struct{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	var out []Result
	for _, r := range h.results {
		if r.ID > since {
			out = append(out, r)
		}
	}
	return out, h.resSig
}

// Register mounts the REPL endpoints under prefix. Pass "" to mount at root,
// or e.g. "/<token>/repl" to gate it behind the existing token prefix.
func Register(mux *http.ServeMux, prefix string) {
	sub, err := fs.Sub(assets, ".")
	if err != nil {
		panic(err)
	}
	mux.HandleFunc("GET "+prefix+"/{$}", serveAsset(sub, "repl.html", "text/html; charset=utf-8"))
	mux.HandleFunc("GET "+prefix+"/executor.js", serveAsset(sub, "executor.js", "application/javascript; charset=utf-8"))
	mux.HandleFunc("GET "+prefix+"/ping", handlePing)
	mux.HandleFunc("GET "+prefix+"/jsin", handleJSIn)
	mux.HandleFunc("POST "+prefix+"/stdout", handleStdout)
	mux.HandleFunc("POST "+prefix+"/cmd", handleCmd)
	mux.HandleFunc("GET "+prefix+"/outbox", handleOutbox)
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

func handlePing(w http.ResponseWriter, r *http.Request) {
	log.Printf("ping from %s", r.RemoteAddr)
	w.Write([]byte("ok"))
}

func handleJSIn(w http.ResponseWriter, r *http.Request) {
	deadline := time.NewTimer(longPollWait)
	defer deadline.Stop()
	for {
		j, ok, sig := h.tryDequeue()
		if ok {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(j)
			return
		}
		select {
		case <-sig:
		case <-deadline.C:
			w.WriteHeader(http.StatusNoContent)
			return
		case <-r.Context().Done():
			return
		}
	}
}

func handleStdout(w http.ResponseWriter, r *http.Request) {
	var res Result
	if err := json.NewDecoder(r.Body).Decode(&res); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	res.At = time.Now().Unix()
	h.addResult(res)
	w.WriteHeader(http.StatusNoContent)
}

func handleCmd(w http.ResponseWriter, r *http.Request) {
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
	j := h.enqueue(body.Code)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(j)
}

func handleOutbox(w http.ResponseWriter, r *http.Request) {
	since, _ := strconv.Atoi(r.URL.Query().Get("since"))
	deadline := time.NewTimer(longPollWait)
	defer deadline.Stop()
	for {
		results, sig := h.resultsSince(since)
		if len(results) > 0 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(results)
			return
		}
		select {
		case <-sig:
		case <-deadline.C:
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("[]"))
			return
		case <-r.Context().Done():
			return
		}
	}
}
