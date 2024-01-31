package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ScreenMode is the geometry mode from SET SCREEN.
type ScreenMode string

const (
	// ScreenAuto follows the host terminal size.
	ScreenAuto ScreenMode = "AUTO"
	// ScreenDefault pins 80 by 24.
	ScreenDefault ScreenMode = "DEFAULT"
)

// Config holds the session flags and sizes from the constitution defaults.
type Config struct {
	Talk        bool
	Intensity   bool
	Bell        bool
	Heading     bool
	Safety      bool
	Help        bool
	History     bool
	Scoreboard  bool
	Menus       bool
	Exact       bool
	Deleted     bool
	DoHistory   bool
	Echo        bool
	Step        bool
	Carry       bool
	Confirm     bool
	Delimiters  bool
	DelimChars  string
	Escape      bool
	Console     bool
	Alternate   bool
	AlternateTo string
	Debug       bool
	Print       bool
	Device      string // SCREEN or PRINT
	Fixed       bool
	Color       string
	ColorOn     bool   // SET COLOR ON/OFF — apply Color when presenting
	PrinterTo   string // SET PRINTER TO path; empty = default stdout sink
	Format      string
	Margin      int
	Decimals    int
	Default     string
	Path        string
	HistoryTo   int
	Typeahead   int
	MemoWidth   int
	Status      bool
	Screen      ScreenMode
	DateOrder   string // AMERICAN, ANSI, …
	Century     bool
	Unique      bool // SET UNIQUE for next INDEX ON
	Fields      bool // SET FIELDS ON/OFF (session default mirrors areas)
	Exclusive   bool // SET EXCLUSIVE
	Encryption  bool // SET ENCRYPTION (accepted; local no-op)
	CatalogOn   bool // SET CATALOG ON/OFF
	// Host adaptations (not Ashton-Tate verbs) — set from environment in New.
	NoAltScreen bool // GOBI_NO_ALTSCREEN — stay on the primary buffer
}

// New returns the constitution defaults, then applies host environment overrides.
func New() *Config {
	c := &Config{
		Talk:       true,
		Intensity:  true,
		Bell:       true,
		Heading:    true,
		Safety:     true,
		Help:       true,
		History:    true,
		Scoreboard: true,
		Menus:      true,
		Exact:      false,
		Deleted:    false,
		DoHistory:  false,
		Echo:       false,
		Step:       false,
		Carry:      false,
		Confirm:    false,
		Delimiters: false,
		DelimChars: ":",
		Escape:     true,
		Console:    true,
		Alternate:  false,
		Debug:      false,
		Print:      false,
		Device:     "SCREEN",
		Fixed:      false,
		Margin:     0,
		Decimals:   0,
		Default:    ".",
		Path:       "",
		HistoryTo:  20,
		Typeahead:  20,
		MemoWidth:  50,
		Status:     stdoutIsTerminal(),
		Screen:     ScreenAuto,
		DateOrder:  "AMERICAN",
		Century:    false,
	}
	c.ApplyHostEnv()
	return c
}

// ApplyHostEnv reads optional GOBI_* variables for modern hosts (UTF-8 paths OK).
//
//	GOBI_DEFAULT       initial SET DEFAULT path
//	GOBI_PATH          initial SET PATH (os list separator)
//	GOBI_SCREEN        AUTO | DEFAULT
//	GOBI_STATUS        ON | OFF | 1 | 0 | true | false
//	GOBI_SCOREBOARD    ON | OFF | …
//	GOBI_NO_ALTSCREEN  ON — skip alternate screen (broken terminals / some CI)
func (c *Config) ApplyHostEnv() {
	if c == nil {
		return
	}
	if v := strings.TrimSpace(os.Getenv("GOBI_DEFAULT")); v != "" {
		c.Default = filepath.Clean(v)
	}
	if v := strings.TrimSpace(os.Getenv("GOBI_PATH")); v != "" {
		c.Path = v
	}
	switch strings.ToUpper(strings.TrimSpace(os.Getenv("GOBI_SCREEN"))) {
	case "AUTO":
		c.Screen = ScreenAuto
	case "DEFAULT":
		c.Screen = ScreenDefault
	}
	if v, ok := envBool("GOBI_STATUS"); ok {
		c.Status = v
	}
	if v, ok := envBool("GOBI_SCOREBOARD"); ok {
		c.Scoreboard = v
	}
	if v, ok := envBool("GOBI_NO_ALTSCREEN"); ok {
		c.NoAltScreen = v
	}
	if v, ok := envBool("GOBI_COLOR"); ok {
		c.ColorOn = v
	}
}

func envBool(name string) (bool, bool) {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return false, false
	}
	switch strings.ToUpper(v) {
	case "1", "ON", "TRUE", "YES", "Y":
		return true, true
	case "0", "OFF", "FALSE", "NO", "N":
		return false, true
	}
	if b, err := strconv.ParseBool(v); err == nil {
		return b, true
	}
	return false, false
}

func stdoutIsTerminal() bool {
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
