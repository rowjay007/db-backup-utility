package util

import "os"

// MergeEnv merges new env entries into the current process environment.
func MergeEnv(extra []string) []string {
	env := append([]string{}, os.Environ()...)
	env = append(env, extra...)
	return env
}
