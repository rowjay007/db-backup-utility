package db

import "os"

func stderrSink() *os.File {
	return os.Stderr
}
