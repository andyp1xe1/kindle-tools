// Command wallpapers runs the Kindle wallpaper manager HTTP server.
package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"

	"wallpapers/internal/server"
	"wallpapers/internal/wallpapers"
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
