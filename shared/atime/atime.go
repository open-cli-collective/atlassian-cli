package atime

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// AtlassianTime wraps time.Time with a custom JSON unmarshaler that handles
// the three timestamp formats found across Atlassian Cloud APIs:
//
//   - ISO 8601 string (Jira/Confluence core): "2023-12-04T10:00:00.000+0000"
//   - Integer epoch millis (webhooks): 1461049397396
//   - Decimal epoch seconds.nanos (Automation/Jackson jsr310): 1701482354.625000000
//
// Use *AtlassianTime in struct fields with `json:",omitempty"` to suppress nil values.
type AtlassianTime struct {
	time.Time
}

var isoLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02T15:04:05.000Z",
	"2006-01-02T15:04:05.000-0700",
}

func (t *AtlassianTime) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}

	if data[0] == '"' {
		s := strings.Trim(string(data), `"`)
		if s == "" {
			return nil
		}
		for _, layout := range isoLayouts {
			if parsed, err := time.Parse(layout, s); err == nil {
				t.Time = parsed
				return nil
			}
		}
		return fmt.Errorf("AtlassianTime: cannot parse ISO string %q", s)
	}

	s := string(data)
	secStr, nanoStr, hasDot := strings.Cut(s, ".")
	if hasDot {
		sec, err := strconv.ParseInt(secStr, 10, 64)
		if err != nil {
			return fmt.Errorf("AtlassianTime: invalid seconds in %q: %w", s, err)
		}
		nanoStr = (nanoStr + "000000000")[:9]
		nanos, err := strconv.ParseInt(nanoStr, 10, 64)
		if err != nil {
			return fmt.Errorf("AtlassianTime: invalid nanos in %q: %w", s, err)
		}
		t.Time = time.Unix(sec, nanos).UTC()
		return nil
	}

	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("AtlassianTime: cannot unmarshal %q: %w", s, err)
	}
	// Epoch seconds are ≤10 digits through 2286; epoch millis are 13 digits for modern dates.
	// 11-12 digit values are ambiguous but don't occur in Atlassian API responses.
	if len(s) <= 10 {
		t.Time = time.Unix(n, 0).UTC()
	} else {
		t.Time = time.UnixMilli(n).UTC()
	}
	return nil
}

func (t AtlassianTime) String() string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}
