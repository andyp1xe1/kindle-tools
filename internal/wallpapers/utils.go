package wallpapers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"wallpapers/internal/kindle"
)

func decodeName(r *http.Request, field string) (string, error) {
	body := map[string]string{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return "", err
	}
	name := strings.TrimSpace(body[field])
	if !presetNameRE.MatchString(name) {
		return "", errors.New("bad preset name")
	}
	return name, nil
}

func httpErr(w http.ResponseWriter, err error, code int) { http.Error(w, err.Error(), code) }

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// withRW wraps kindle.WithRW with the dev-mode escape hatch: in dev we run
// off a writable working tree, so the mntroot dance is skipped.
func withRW(fn func() error) error {
	if devMode {
		return fn()
	}
	return kindle.WithRW(fn)
}
