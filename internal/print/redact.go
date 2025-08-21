package print

// Redact masks secrets for "show" output.
func Redact(s string) string {
	if len(s) == 0 {
		return ""
	}
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + "****" + s[len(s)-2:]
}
