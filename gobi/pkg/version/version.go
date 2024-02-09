// Package version holds the Gobi product identity shown in the UI and VERSION().
package version

// Version is the product version returned by VERSION() (no leading "v").
const Version = "1.0"

// Product is the VERSION() product token (uppercase, dBase-style).
const Product = "GOBI"

// DisplayName is the on-screen brand (never "dBASE").
const DisplayName = "Gobi"

// String returns "GOBI 1.0" for VERSION().
func String() string {
	return Product + " " + Version
}
