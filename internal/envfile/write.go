package envfile

import (
	"bufio"
	"fmt"
	"os"
	"sort"
)

// WriteEnv writes key→value pairs to path, merging if the file exists.
func WriteEnv(path string, kv map[string]string) error {
	out := map[string]string{}

	// load existing
	if f, err := os.Open(path); err == nil {
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := sc.Text()
			if len(line) == 0 || line[0] == '#' {
				continue
			}
			if kvp := splitLine(line); kvp != nil {
				out[kvp[0]] = kvp[1]
			}
		}
		_ = f.Close()
	}

	// merge / overwrite
	for k, v := range kv {
		out[k] = v
	}

	// write
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	// stable sort for diff-friendly files
	keys := make([]string, 0, len(out))
	for k := range out {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if _, err := fmt.Fprintf(f, "%s=%s\n", k, out[k]); err != nil {
			return err
		}
	}
	return nil
}

// naive split key=value
func splitLine(l string) []string {
	for i := 0; i < len(l); i++ {
		if l[i] == '=' {
			return []string{l[:i], l[i+1:]}
		}
	}
	return nil
}
