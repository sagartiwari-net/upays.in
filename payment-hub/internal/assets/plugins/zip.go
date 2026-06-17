package plugins

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func ZipDirectory(dir string) ([]byte, error) {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return nil, err
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	base := filepath.Clean(dir)

	err = filepath.Walk(base, func(path string, fi os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if fi.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(base, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, ".") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		fw, err := zw.Create(rel)
		if err != nil {
			return err
		}
		_, err = io.Copy(fw, bytes.NewReader(data))
		return err
	})
	if err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func FindWooCommercePluginDir() string {
	candidates := []string{
		filepath.Join("..", "plugins", "woocommerce", "upipays-wc"),
		filepath.Join("plugins", "woocommerce", "upipays-wc"),
	}
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
	}
	return ""
}
