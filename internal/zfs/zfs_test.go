package zfs

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestNormalizeAndValid(t *testing.T) {
	for in, want := range map[string]string{"zfs-2.4.4": "2.4.4", "v2.3.9": "2.3.9", " 2.2.11 ": "2.2.11", "": ""} {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
	for _, ok := range []string{"", "2.4.4", "2.10.0"} {
		if !Valid(ok) {
			t.Errorf("Valid(%q) = false", ok)
		}
	}
	for _, bad := range []string{"2.4", "2.4.4-rc1", "2.4.4; rm -rf /", "latest", "$(id)"} {
		if Valid(bad) {
			t.Errorf("Valid(%q) = true", bad)
		}
	}
}

func TestVersions(t *testing.T) {
	hits := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Write([]byte(`[
			{"tag_name":"zfs-2.3.9","prerelease":false,"draft":false},
			{"tag_name":"zfs-2.4.4","prerelease":false,"draft":false},
			{"tag_name":"zfs-2.5.0-rc1","prerelease":true,"draft":false},
			{"tag_name":"zfs-2.10.0","prerelease":false,"draft":false},
			{"tag_name":"zfs-2.4.5","prerelease":false,"draft":true}]`))
	}))
	defer ts.Close()

	l := NewLister()
	l.releasesURL = ts.URL
	for i := 0; i < 2; i++ {
		got, err := l.Versions()
		if err != nil {
			t.Fatal(err)
		}
		if want := []string{"2.10.0", "2.4.4", "2.3.9"}; !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}
	if hits != 1 {
		t.Errorf("expected cached second call, got %d requests", hits)
	}
}
