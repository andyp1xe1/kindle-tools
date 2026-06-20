package wallpapers

import (
	"fmt"

	"github.com/andyp1xe1/kindle-tools/internal/kindle"
)

type deviceInfo struct {
	Model     string `json:"model"`
	W         int    `json:"w"`
	H         int    `json:"h"`
	Bpp       int    `json:"bpp"`
	Grayscale bool   `json:"grayscale"`
}

func gatherDevice() deviceInfo {
	w, h, bpp, gray := kindle.ParseEips()
	if w == 0 || h == 0 {
		w, h, bpp, gray = 1072, 1448, 8, true
	}
	name := ""
	if m, ok := kindle.ModelFromSerial(); ok {
		name = m.Name
	}
	if name == "" {
		name = fmt.Sprintf("Kindle (%d×%d)", w, h)
	}
	return deviceInfo{name, w, h, bpp, gray}
}
