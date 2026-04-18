package resolve

import "testing"

func TestLooksLikeProjectKey(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"MON", true},
		{"CAPONE", true},
		{"A1_B2", true},
		{"M", false},            // single char — too short
		{"MONPROJLONG1", false}, // 12 chars — too long
		{"mon", false},          // lowercase
		{"MON-1", false},        // hyphen
		{"Platform Development", false},
	}
	for _, tc := range cases {
		if got := looksLikeProjectKey(tc.in); got != tc.want {
			t.Errorf("looksLikeProjectKey(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestLooksLikeNumeric(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"0", true},
		{"125", true},
		{"", false},
		{"12a", false},
		{"-5", false},
	}
	for _, tc := range cases {
		if got := looksLikeNumeric(tc.in); got != tc.want {
			t.Errorf("looksLikeNumeric(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestLooksLikeAccountID(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"60e09bae7fcd820073089249", true}, // 24-char hex
		{"557058:295fe89c-10c2-4b0c-ba84-a4dd14ea7729", true},
		{"Rusty Hall", false},       // contains space
		{"me", false},               // too short
		{"short", false},            // <16 chars
		{"rian@monitapp.io", false}, // contains @
	}
	for _, tc := range cases {
		if got := looksLikeAccountID(tc.in); got != tc.want {
			t.Errorf("looksLikeAccountID(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
