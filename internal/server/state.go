package server

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type presetInfo struct {
	Name  string `json:"name"`
	Files int    `json:"files"`
}

type appState struct {
	Device    deviceInfo   `json:"device"`
	Presets   []presetInfo `json:"presets"`
	Active    string       `json:"active"`    // active preset name, "" if none
	Unclaimed int          `json:"unclaimed"` // file count in screensaver/ if it's a regular dir
}

func handleState(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, gatherState())
}

func gatherState() appState {
	s := appState{Device: gatherDevice(), Presets: []presetInfo{}}
	wp := filepath.Join(blanket, "wallpapers")
	if entries, err := os.ReadDir(wp); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			s.Presets = append(s.Presets, presetInfo{e.Name(), countPNGs(filepath.Join(wp, e.Name()))})
		}
	}
	sort.Slice(s.Presets, func(i, j int) bool { return s.Presets[i].Name < s.Presets[j].Name })

	ss := filepath.Join(blanket, "screensaver")
	if fi, err := os.Lstat(ss); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			s.Active = activeFromLink(ss)
		} else if fi.IsDir() {
			s.Unclaimed = countPNGs(ss)
		}
	}
	return s
}

func countPNGs(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() && ssNameRE.MatchString(e.Name()) {
			n++
		}
	}
	return n
}

// activeFromLink resolves the screensaver symlink to a preset name, accepting
// both relative ("wallpapers/foo") and absolute targets.
func activeFromLink(ssPath string) string {
	target, err := os.Readlink(ssPath)
	if err != nil {
		return ""
	}
	abs := target
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(filepath.Dir(ssPath), target)
	}
	rel, err := filepath.Rel(filepath.Join(blanket, "wallpapers"), abs)
	if err != nil || strings.HasPrefix(rel, "..") || strings.ContainsRune(rel, filepath.Separator) {
		return ""
	}
	return rel
}
