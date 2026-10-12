package builder

import (
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

var vercmpCases = []struct {
	a, b string
	want int
}{
	{"1.0", "1.0", 0},
	{"1.0", "1.1", -1},
	{"1.10", "1.9", 1},
	{"1.0.0", "1.0", 1},
	{"1.0", "1.0.", -1},
	{"1.0a", "1.0", -1},
	{"1.0alpha", "1.0beta", -1},
	{"1.0", "1.0rc1", 1},
	{"1.01", "1.1", 0},
	{"1.0-1", "1.0-2", -1},
	{"1.0-2", "1.0", 0}, // a missing pkgrel matches any
	{"1:1.0", "2.0", 1},
	{"0:1.0", "1.0", 0},
	{"1.0+", "1.0.", 0}, // separators only count by length
	{"2.4.4_7.2.9.arch1.1-1", "2.4.4_7.2.8.arch1.2-1", 1},
	{"7.2.9.arch1.1-1", "7.2.10.arch1.1-1", -1},
	{"6.18.55.1-1", "6.18.9.1-1", 1},
	{"2.4.4", "2.4.4-1", 0},
	{"a", "1", -1},
	{"1", "a", 1},
}

func TestVercmp(t *testing.T) {
	for _, c := range vercmpCases {
		if got := Vercmp(c.a, c.b); got != c.want {
			t.Errorf("Vercmp(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
		if got := Vercmp(c.b, c.a); got != -c.want {
			t.Errorf("Vercmp(%q, %q) = %d, want %d", c.b, c.a, got, -c.want)
		}
	}
}

// The port must agree with pacman's own vercmp, also on pairs nobody thought of in the table above.
func TestVercmpMatchesPacman(t *testing.T) {
	if _, err := exec.LookPath("vercmp"); err != nil {
		t.Skip("vercmp not available")
	}
	versions := []string{"1", "1.0", "1.0.", "1.0a", "1.0b1", "1.0rc", "1.0-1", "1.0-1.1", "1:0.9", "1.0~1", "1..0",
		"2.4.4", "2.4.10", "2.4.4_7.2.9.arch1.1-1", "2.4.4_7.2.9.arch1.2-1", "2.4.5_7.2.8.arch1.2-1", "7.2.9.arch1-1",
		"6.18.55-1", "6.18.55.1", "abc", "a1b2", "1a", "01", "0", "", "1.0_rc1", "1.0.0.0"}
	for _, a := range versions {
		for _, b := range versions {
			out, err := exec.Command("vercmp", a, b).Output()
			if err != nil {
				t.Fatalf("vercmp %q %q: %v", a, b, err)
			}
			want, _ := strconv.Atoi(strings.TrimSpace(string(out)))
			if got := Vercmp(a, b); got != want {
				t.Errorf("Vercmp(%q, %q) = %d, pacman says %d", a, b, got, want)
			}
		}
	}
}
