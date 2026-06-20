package kindle

import "strings"

// Kindle device identification by /proc/usid serial prefix.
// Reference: https://wiki.mobileread.com/wiki/Kindle_Serial_Numbers

type Model struct {
	Name     string
	W, H     int
	Prefixes []string
}

var Models = []Model{
	// --- 11th gen (2021-2022) ----------------------------------------------
	{"Kindle (11th gen)", 1072, 1448, []string{
		"G002AQ", "G002AP", "G092AP",
		"G092AQ",
	}},
	{"Paperwhite 5", 1236, 1648, []string{
		"G001LG", "G001PX", "G8S1PX",
		"G002BH", "G002DK", "G8S2DK",
		"G0021A", "G00219", "G8S219",
	}},
	{"Scribe", 1860, 2480, []string{
		"G00227", "G0023M", "G0923M", "G0023L",
	}},

	// --- 10th gen (2018-2019) ----------------------------------------------
	{"Paperwhite 4", 1072, 1448, []string{
		"G000PP", "G8S0PP", "G000T6", "G8S0T6", "G000T1", "G000T2",
		"G00102", "G000T3",
		"G0016T", "G8S16T", "G0016Q", "G8S16Q",
		"G0016U", "G0016V", "G8S16V",
		"G00103", "G0016R", "G0016S",
	}},
	{"Kindle Basic 3", 600, 800, []string{
		"G0910L", "G090WH", "G090VB", "G090WF",
	}},
	{"Oasis 3", 1264, 1680, []string{
		"G0011L", "G000WQ", "G000WN", "G000WM", "G000WL", "G000WP",
	}},

	// --- 9th gen (2017) ----------------------------------------------------
	{"Oasis 2", 1264, 1680, []string{
		"G000P8", "G000S1", "G000SA", "G000S2", "G000P1",
	}},

	// --- 8th gen (2016) ----------------------------------------------------
	{"Kindle Basic 2", 600, 800, []string{"G000K9", "G000KA"}},
	{"Oasis", 1072, 1448, []string{
		"G0B0GC", "G0B0GD", "G0B0GR", "G0B0GU", "G0B0GT",
	}},

	// --- 7th gen (2014-2015) -----------------------------------------------
	{"Paperwhite 3", 1072, 1448, []string{
		"G090G1", "G090G2", "G090G4", "G090G5", "G090G6", "G090G7",
		"G090KB", "G090KC", "G090KE", "G090KF", "G090LK", "G090LL",
	}},
	{"Voyage", 1072, 1448, []string{
		"B013", "9013", "B054", "9054", "B053", "9053", "B02A", "B052", "9052", "B04F",
	}},
	{"Kindle Basic", 600, 800, []string{"B0C6", "90C6", "B0DD", "90DD"}},

	// --- 6th gen (2013) ----------------------------------------------------
	{"Paperwhite 2", 758, 1024, []string{
		"B0D4", "90D4", "B05A", "905A", "B0D5", "90D5", "B0D6", "90D6",
		"B0D7", "90D7", "B0D8", "90D8", "B0F2", "90F2",
		"B017", "9017", "B060", "9060", "B062", "9062", "B05F", "905F", "B061", "9061",
	}},

	// --- 5th gen (2012) ----------------------------------------------------
	{"Paperwhite", 758, 1024, []string{
		"B024", "B01B", "B020", "B01C", "B01D", "B01F",
	}},

	// --- 4th gen (2011) ----------------------------------------------------
	{"Kindle Touch", 600, 800, []string{"B00F", "B011", "B010", "B012"}},
	{"Kindle 4 NoTouch", 600, 800, []string{"B00E", "B023", "9023"}},

	// --- 3rd gen (2010) ----------------------------------------------------
	{"Kindle 3", 600, 800, []string{"B008", "B006", "B00A"}},

	// --- 2nd gen (2009) ----------------------------------------------------
	{"Kindle DX", 824, 1200, []string{"B004", "B005", "B009"}},
	{"Kindle 2", 600, 800, []string{"B002", "B003"}},

	// --- 1st gen (2007) ----------------------------------------------------
	{"Kindle", 600, 800, []string{"B001", "B101"}},
}

// MatchModel returns the model whose prefix is the longest hit on serial.
func MatchModel(serial string) (Model, bool) {
	best := 0
	var hit Model
	for _, m := range Models {
		for _, p := range m.Prefixes {
			if len(p) > best && strings.HasPrefix(serial, p) {
				best = len(p)
				hit = m
			}
		}
	}
	return hit, best > 0
}
