package main

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

func TestStaticPagesUseLocalScriptAssets(t *testing.T) {
	err := fs.WalkDir(staticFS, "static", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".html" {
			return nil
		}
		b, err := staticFS.ReadFile(path)
		if err != nil {
			return err
		}
		html := string(b)
		for _, forbidden := range []string{"cdn.jsdelivr.net", `script src="https://`, `script src="http://`} {
			if strings.Contains(html, forbidden) {
				t.Fatalf("%s contains external script/resource reference %q", path, forbidden)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
