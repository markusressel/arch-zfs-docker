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
