// Package web embeds the static UI assets served by the wallpapers server.
package web

import "embed"

//go:embed app.js index.html style.css
var FS embed.FS
