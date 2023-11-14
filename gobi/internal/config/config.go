package config

import (
	"os"
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
}

// New returns the constitution defaults. Status is on when os.Stdout is a terminal.
func New() *Config {
	return &Config{
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
	}
}

func stdoutIsTerminal() bool {
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
