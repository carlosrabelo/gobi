// Package version holds the Gobi product identity shown in the UI and VERSION().
package version

// Version is the semver-like product version (no leading "v").
const Version = "0.1.0"

// Product is the display name (never "dBASE" on screen).
const Product = "Gobi"

// String returns "Gobi 0.1.0" for VERSION() (not status-bar chrome).
func String() string {
	return Product + " " + Version
}
