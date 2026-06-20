// Command jsrepl runs the reverse JS REPL as a standalone HTTP server.
// The on-Kindle scriptlet loads /executor.js to install a silent long-poll
// executor that runs jobs sent from the laptop UI at /{TOK}/.
package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"

	"github.com/andyp1xe1/kindle-tools/internal/jsrepl"
	"github.com/andyp1xe1/kindle-tools/internal/server"
)

func main() {
	dev := flag.Bool("dev", false, "skip firewall opening so the binary runs on a non-Kindle host")
	flag.Parse()

	s := server.New(jsrepl.Name, jsrepl.Port, *dev)
	jsrepl.Register(s)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	log.Fatal(s.Run(ctx))
}
