package atime

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestUnmarshalJSON(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		{
			name:  "ISO 8601 RFC3339",
			input: `"2023-12-04T09:00:00Z"`,
			want:  time.Date(2023, 12, 4, 9, 0, 0, 0, time.UTC),
		},
		{
			name:  "ISO 8601 .000Z variant",
			input: `"2023-12-04T10:00:00.000Z"`,
			want:  time.Date(2023, 12, 4, 10, 0, 0, 0, time.UTC),
		},
		{
			name:  "ISO 8601 +0000 variant",
			input: `"2023-12-04T10:00:00.000+0000"`,
			want:  time.Date(2023, 12, 4, 10, 0, 0, 0, time.UTC),
		},
		{
			name:  "ISO 8601 -0700 variant",
			input: `"2023-12-04T03:00:00.000-0700"`,
			want:  time.Date(2023, 12, 4, 3, 0, 0, 0, time.FixedZone("-0700", -7*60*60)),
		},
		{
			name:  "fractional epoch seconds.nanos",
			input: `1701482354.625000000`,
			want:  time.Unix(1701482354, 625000000).UTC(),
		},
		{
			name:  "fractional epoch short nanos",
			input: `1701482354.625`,
			want:  time.Unix(1701482354, 625000000).UTC(),
		},
		{
			name:  "integer epoch millis 13 digits",
			input: `1701680400000`,
			want:  time.UnixMilli(1701680400000).UTC(),
		},
		{
			name:  "integer epoch seconds 10 digits",
			input: `1701482354`,
			want:  time.Unix(1701482354, 0).UTC(),
		},
		{
			name:  "null",
			input: `null`,
			want:  time.Time{},
		},
		{
			name:  "empty string",
			input: `""`,
			want:  time.Time{},
		},
		{
			name:    "invalid boolean",
			input:   `true`,
			wantErr: true,
		},
		{
			name:    "unparseable ISO string",
			input:   `"not-a-date"`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var at AtlassianTime
			err := at.UnmarshalJSON([]byte(tt.input))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !at.Equal(tt.want) {
				t.Errorf("got %v, want %v", at.Time, tt.want)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	t.Parallel()
	original := AtlassianTime{Time: time.Date(2023, 12, 4, 9, 0, 0, 0, time.UTC)}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var restored AtlassianTime
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !restored.Equal(original.Time) {
		t.Errorf("round-trip failed: got %v, want %v", restored.Time, original.Time)
	}
}

func TestNilPointerOmitEmpty(t *testing.T) {
	t.Parallel()
	type wrapper struct {
		Created *AtlassianTime `json:"created,omitempty"`
		Name    string         `json:"name"`
	}
	w := wrapper{Name: "test"}
	data, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(data)
	if strings.Contains(s, "created") {
		t.Errorf("nil *AtlassianTime should be omitted, got: %s", s)
	}
}

func TestString(t *testing.T) {
	t.Parallel()
	at := AtlassianTime{Time: time.Date(2023, 12, 4, 9, 0, 0, 0, time.UTC)}
	if at.String() != "2023-12-04T09:00:00Z" {
		t.Errorf("String() = %q", at.String())
	}

	zero := AtlassianTime{}
	if zero.String() != "" {
		t.Errorf("zero String() = %q, want empty", zero.String())
	}
}
