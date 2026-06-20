// Package wallpapers implements the Kindle wallpaper manager.
//
// Layout under /usr/share/blanket:
//
//	wallpapers/<preset>/bg_ssNN.png       — user-owned preset directories
//	screensaver -> wallpapers/<preset>    — symlink the OS reads from
//
// Activating a preset swaps the symlink atomically; uploads land inside the
// preset dir directly, so they show up on the next screen-off.
package wallpapers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/andyp1xe1/kindle-tools/internal/server"
	"github.com/andyp1xe1/kindle-tools/internal/wallpapers/web"
)

var (
	devMode bool
	blanket string

	ssNameRE     = regexp.MustCompile(`^bg_ss\d{2,3}\.png$`)
	presetNameRE = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}$`)
)

// Register initializes the blanket layout and mounts the app routes under s.Token.
func Register(s *server.Server, dev bool) error {
	devMode = dev
	if dev {
		blanket = devDir
	} else {
		blanket = blanketDir
	}
	if err := withRW(initLayout); err != nil {
		return fmt.Errorf("init: %w", err)
	}

	pfx := "/" + s.Token
	s.HandleProtected("/", http.StripPrefix(pfx, http.FileServer(http.FS(web.FS))))
	s.HandleFuncProtected("GET /api/state", handleState)
	s.HandleFuncProtected("POST /api/claim", handleClaim)
	s.HandleFuncProtected("POST /api/presets", handleCreatePreset)
	s.HandleFuncProtected("DELETE /api/presets/{name}", handleDeletePreset)
	s.HandleFuncProtected("POST /api/presets/{name}/activate", handleActivatePreset)
	s.HandleFuncProtected("POST /api/presets/{name}/rename", handleRenamePreset)
	s.HandleFuncProtected("GET /api/presets/{name}/files", handleListFiles)
	s.HandleFuncProtected("GET /api/presets/{name}/files/{file}", handleGetFile)
	s.HandleFuncProtected("POST /api/presets/{name}/files", handleUploadFile)
	s.HandleFuncProtected("DELETE /api/presets/{name}/files/{file}", handleDeleteFile)
	s.HandleFuncProtected("POST /api/import", handleImport)
	return nil
}

func initLayout() error {
	return os.MkdirAll(filepath.Join(blanket, "wallpapers"), 0755)
}
