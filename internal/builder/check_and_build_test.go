package builder

import (
	"strings"
	"testing"
)

// Concurrent jobs share the pacman cache on /repo. A recursive chown of the whole cache by one
// job breaks the sandboxed downloads of the others ("could not open file ...part: Permission denied").
func TestBuildScriptDoesNotChownPacmanCache(t *testing.T) {
	for _, line := range strings.Split(CheckAndBuildScript, "\n") {
		if strings.Contains(line, "chown") && strings.Contains(line, `"$CACHE_DIR"`) {
			t.Errorf("script chowns the shared cache directory: %s", line)
		}
	}
}

// sudo -E keeps HOME=/root, where the build user cannot create a gpg keyring: the key import and
// makepkg's signature check would fail.
func TestBuildScriptGivesTheBuildUserItsOwnHome(t *testing.T) {
	_, build, ok := strings.Cut(CheckAndBuildScript, "sudo -E -u build bash << 'EOF_BUILD'")
	if !ok {
		t.Fatal("build user section not found")
	}
	if !strings.Contains(build[:strings.Index(build, "gpg --batch")], "export HOME=/home/build") {
		t.Error("HOME must point to the build user's home before gpg runs")
	}
}
