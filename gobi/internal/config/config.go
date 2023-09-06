package config

// Config stores the global environment settings for Gobi.
// These correspond directly to the dBase II 'SET' parameters.
type Config struct {
	Talk       bool   // Echo back commands execution result / records count
	Intensity  bool   // Use reverse video/highlighting in TUI input fields
	Bell       bool   // Trigger system bell sound on validation errors
	DefaultDir string // Default working directory path for databases and scripts
	ScreenAuto bool   // Adapt screen geometry to the real terminal size (SET SCREEN AUTO/DEFAULT)
	Exact      bool   // Exact string comparison instead of dBase II prefix matching (SET EXACT)
	Deleted    bool   // Hide records marked for deletion from scans and navigation (SET DELETED)
}

// New returns a Config initialized with dBase II default settings.
func New() *Config {
	return &Config{
		Talk:       true,
		Intensity:  true,
		Bell:       true,
		DefaultDir: ".",
		ScreenAuto: true,
	}
}
