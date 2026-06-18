// Command wallpapers runs the Kindle wallpaper manager HTTP server.
package main

import (
	"flag"
	"log"

	"wallpapers/internal/server"
)

func main() {
	var (
		dev  = flag.Bool("dev", false, "use ./dev as the blanket root, skip mntroot")
		port = flag.Int("port", 6969, "http port")
	)
	flag.Parse()
	log.Fatal(server.Run(server.Config{Port: *port, DevMode: *dev}))
}
