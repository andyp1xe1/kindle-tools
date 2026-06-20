// Command wallpapers runs the Kindle wallpaper manager HTTP server.
package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"

	"github.com/andyp1xe1/kindle-tools/internal/server"
	"github.com/andyp1xe1/kindle-tools/internal/wallpapers"
)

func main() {
	dev := flag.Bool("dev", false, "use ./dev as the blanket root, skip mntroot")
	flag.Parse()

	s := server.New(wallpapers.Name, wallpapers.Port, *dev)
	if err := wallpapers.Register(s, *dev); err != nil {
		log.Fatal(err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	log.Fatal(s.Run(ctx))
}
