package version

var (
	// These three values are injected via -ldflags at build time.
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)
