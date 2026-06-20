package kindle

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// ParseEips returns screen geometry by parsing `eips -i` output.
// All zeros if eips is missing or unreadable. Forces portrait orientation
// (some firmwares report landscape).
func ParseEips() (w, h, bpp int, gray bool) {
	out, err := exec.Command("eips", "-i").Output()
	if err != nil {
		return 0, 0, 0, false
	}
	m := map[string]int{}
	fields := strings.Fields(string(out))
	for i := 0; i < len(fields)-1; i++ {
		if k, ok := strings.CutSuffix(fields[i], ":"); ok {
			if v, err := strconv.Atoi(fields[i+1]); err == nil {
				m[k] = v
			}
		}
	}
	w, h = m["xres"], m["yres"]
	if w > h {
		w, h = h, w
	}
	// All Kindles are e-ink; assume grayscale unless eips explicitly says 0.
	gray = true
	if v, ok := m["grayscale"]; ok {
		gray = v == 1
	}
	return w, h, m["bits_per_pixel"], gray
}

// ModelFromSerial reads /proc/usid and returns the matched model, or
// (Model{}, false) if the serial is missing or unknown.
func ModelFromSerial() (Model, bool) {
	data, err := os.ReadFile("/proc/usid")
	if err != nil {
		return Model{}, false
	}
	return MatchModel(strings.TrimSpace(string(data)))
}
