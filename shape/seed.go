package shape

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// seedIgnored starts a freshly materialized check tree from the build outputs of the shaping
// checkout: every git-ignored path of root (dependency and build folders, caches) is copied
// into dir, copy-on-write where the file system supports it, so setup and compile run
// incrementally. Nothing tracked is touched: a path that already exists in dir keeps the tree's
// content, and neither .git nor .kogen (which holds what Kogen itself writes) is copied. It is
// best effort: a failure only means the tree starts cold. It returns the number of paths
// seeded and the time it took.
func seedIgnored(root, dir string) (int, time.Duration) {
	start := time.Now()
	ls := exec.Command("git", "ls-files", "--others", "--ignored", "--exclude-standard", "--directory", "-z")
	ls.Dir = root
	out, err := ls.Output()
	if err != nil {
		return 0, time.Since(start)
	}
	seeded := 0
	for _, p := range bytes.Split(out, []byte{0}) {
		rel := strings.TrimSuffix(string(p), "/")
		if rel == "" || rel == ".git" || rel == ".kogen" || strings.HasPrefix(rel, ".git/") || strings.HasPrefix(rel, ".kogen/") {
			continue
		}
		src, dst := filepath.Join(root, rel), filepath.Join(dir, rel)
		if _, err := os.Lstat(dst); err == nil {
			continue
		}
		if os.MkdirAll(filepath.Dir(dst), 0o755) != nil {
			continue
		}
		if cloneCopy(src, dst) {
			seeded++
		}
	}
	return seeded, time.Since(start)
}

// cloneCopy copies src to the absent dst, cloning where possible and otherwise copying.
func cloneCopy(src, dst string) bool {
	attempts := [][]string{{"-a", "--reflink=auto"}, {"-a"}}
	if runtime.GOOS == "darwin" {
		attempts = [][]string{{"-cRp"}, {"-Rp"}}
	}
	for _, flags := range attempts {
		if exec.Command("cp", append(flags, "--", src, dst)...).Run() == nil {
			return true
		}
		os.RemoveAll(dst)
	}
	return false
}
