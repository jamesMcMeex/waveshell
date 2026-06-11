package update

import (
	"fmt"
	"strings"
)

func (m Model) View() string {
	if m.Library.Scanning {
		var b strings.Builder
		b.WriteString("Scanning library...\n")
		if m.Library.ScanTotal > 0 {
			fmt.Fprintf(&b, "  %d / %d files processed\n", m.Library.ScanProcessed, m.Library.ScanTotal)
		} else {
			fmt.Fprintf(&b, "  %d files found\n", m.Library.ScanProcessed)
		}
		if m.Library.ScanCurrent != "" {
			fmt.Fprintf(&b, "  current: %s\n", m.Library.ScanCurrent)
		}
		return b.String()
	}

	if m.Library.ScanComplete {
		return fmt.Sprintf("Scan complete — %d processed, %d skipped\n", m.Library.ScanProcessed, m.Library.ScanSkipped)
	}

	return "waveshell\n"
}
