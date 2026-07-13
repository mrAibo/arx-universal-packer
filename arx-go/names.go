package main

import (
	"os"
	"path/filepath"
	"strings"
)

func defaultOutputName(source string) string {
	base := filepath.Base(filepath.Clean(source))
	lower := strings.ToLower(base)
	for _, suffix := range []string{
		".tar.gz",
		".tar.bz2",
		".tar.xz",
		".tar.zst",
		".tgz",
		".tbz2",
		".txz",
		".zip",
		".7z",
		".tar",
	} {
		if strings.HasSuffix(lower, suffix) {
			base = base[:len(base)-len(suffix)]
			break
		}
	}
	if base == "" || base == "." || base == string(os.PathSeparator) {
		return "archive"
	}
	return base
}
