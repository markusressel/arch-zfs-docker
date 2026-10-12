package server

import "testing"

func TestUIURL(t *testing.T) {
	cases := map[string]string{
		":8080":          "http://localhost:8080/ui/",
		"0.0.0.0:8080":   "http://localhost:8080/ui/",
		"[::]:9000":      "http://localhost:9000/ui/",
		"127.0.0.1:8080": "http://127.0.0.1:8080/ui/",
	}
	for in, want := range cases {
		if got := uiURL(in); got != want {
			t.Errorf("uiURL(%q) = %q, want %q", in, got, want)
		}
	}
}
