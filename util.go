package main

import (
	"fmt"
	"unicode/utf8"
)

// humanizeBytes returns the provided number of bytes in humanized form with IEC
// units (aka binary prefixes such as KiB and MiB).
func humanizeBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)

	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++

	}
	return fmt.Sprintf("%.2f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func quoteKey(k []byte) string {
	if utf8.Valid(k) {
		return string(k)
	} else {
		return fmt.Sprintf("%x", k)
	}
}
