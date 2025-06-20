package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Check is a temporary utility function that panics if an error is not nil.
// NOTE: This is a carry-over from the original code. In Step 2 of our plan,
// we will eliminate this function and replace its usage with proper error
// handling (returning errors up the call stack).
func Check(err error) {
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

// ComputeBaseHref calculates the relative path to the site root
// so that CSS/JS links work correctly for pages at any depth.
// For example, a page at /posts/a/b.html would get a BaseHref of "../../".
func ComputeBaseHref(relPath string) string {
	dir := filepath.Dir(relPath)
	if dir == "." {
		return ""
	}
	depth := strings.Count(dir, string(os.PathSeparator)) + 1
	return strings.Repeat("../", depth)
}

