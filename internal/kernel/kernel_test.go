package kernel

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"7.1.8-arch1-3": "7.1.8.arch1-3",
		"7.1.8.arch1-3": "7.1.8.arch1-3",
		" 6.12.5-1 ":    "6.12.5-1",
		"":              "",
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMatches(t *testing.T) {
	if !Matches("7.1.8.arch1.3", "7.1.8-arch1-3") {
		t.Error("expected dotted and hyphenated forms to match")
	}
	if Matches("7.1.8.arch1.2", "7.1.8.arch1-3") {
		t.Error("different pkgrel must not match")
	}
	if Matches("", "7.1.8.arch1-3") {
		t.Error("empty local must not match")
	}
}

func TestParseListing(t *testing.T) {
	html := `<a href="linux-6.9.1.arch1-1-x86_64.pkg.tar.zst">x</a>
<a href="linux-6.9.1.arch1-1-x86_64.pkg.tar.zst.sig">x</a>
<a href="linux-6.10.2.arch2-1-x86_64.pkg.tar.zst">x</a>
<a href="linux-6.9.1.arch1-10-x86_64.pkg.tar.zst">x</a>
<a href="linux-headers-6.9.1.arch1-1-x86_64.pkg.tar.zst">x</a>`
	got := ParseListing(html, "linux")
	want := []string{"6.10.2.arch2-1", "6.9.1.arch1-10", "6.9.1.arch1-1"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestListerVersionsCaches(t *testing.T) {
	hits := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.URL.Path != "/linux-lts/" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Write([]byte(`<a href="linux-lts-6.12.5-1-x86_64.pkg.tar.zst">x</a>`))
	}))
	defer ts.Close()

	l := NewLister()
	l.archiveURL = ts.URL
	for i := 0; i < 2; i++ {
		v, err := l.Versions("lts")
		if err != nil || len(v) != 1 || v[0] != "6.12.5-1" {
			t.Fatalf("unexpected result %v, %v", v, err)
		}
	}
	if hits != 1 {
		t.Errorf("expected 1 upstream request, got %d", hits)
	}
}
