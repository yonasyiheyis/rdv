package print

import (
	"encoding/json"
	"os"
)

// JSON controls whether commands should output JSON.
var JSON bool

// SetJSON toggles JSON output globally.
func SetJSON(on bool) { JSON = on }

// Out prints v as pretty JSON to stdout.
func Out(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
