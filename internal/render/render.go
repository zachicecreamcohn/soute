// Package render formats snapshots into terminal tables.
package render

import (
	"bytes"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/zachicecreamcohn/soute/internal/manifest"
)

// Snapshots renders snapshots as an aligned terminal table.
func Snapshots(s []manifest.Snapshot) string {
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTIME\tSIZE\tDELTA\t%\tTAG\tHASH")
	for _, s := range s {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%+.2f\t%s\t%s\n",
			s.ID,
			ShortTime(s.Timestamp),
			HumanBytes(s.SizeBytes),
			SignedHuman(s.DeltaBytes),
			s.DeltaPct,
			s.Tag,
			ShortHash(s.ContentHash),
		)
	}
	w.Flush()
	return buf.String()
}

// ShortTime renders an RFC3339 timestamp as a local wall-clock time.
func ShortTime(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts
	}
	return t.Local().Format("2006-01-02 15:04:05")
}

// ShortHash truncates a content hash for display.
func ShortHash(h string) string {
	if len(h) > 8 {
		return h[:8]
	}
	return h
}

// HumanBytes formats a byte count in binary units.
func HumanBytes(n int64) string {
	if n < 0 {
		n = -n
	}
	const unit = int64(1024)
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := unit, 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

// SignedHuman formats a signed delta byte count.
func SignedHuman(n int64) string {
	switch {
	case n > 0:
		return "+" + HumanBytes(n)
	case n < 0:
		return "-" + HumanBytes(n)
	default:
		return HumanBytes(0)
	}
}
