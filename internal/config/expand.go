package config

import "os"

// ExpandEnv expands environment variables in a string.
// Supports both $VAR and ${VAR} syntax.
func ExpandEnv(s string) string {
	return os.ExpandEnv(s)
}
