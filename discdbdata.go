package discdb

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func Download() (outputDir string, err error) {
	return downloadCommitFS("TheDiscDb", "data", "56cafdf10b63be23ac2f4875edb4749815b1ab90")
}

// downloadCommitFS downloads a GitHub repository at a specific commit,
// extracts it into a temporary directory, and returns an fs.FS rooted
// at the repository contents.
//
// The returned tempDir should be removed by the caller when finished.
func downloadCommitFS(owner, repo, commit string) (outputDir string, err error) {
	url := fmt.Sprintf(
		"https://github.com/%s/%s/archive/%s.tar.gz",
		owner,
		repo,
		commit,
	)

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err = errors.Join(err, Body.Close())
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github returned %s", resp.Status)
	}

	tempDir, err := os.MkdirTemp("", "githubfs-*")
	if err != nil {
		return "", err
	}

	if err := extractTarGz(resp.Body, tempDir); err != nil {
		return "", errors.Join(err, os.RemoveAll(tempDir))
	}

	return tempDir, nil
}

func extractTarGz(r io.Reader, dst string) (err error) {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer func(gzr *gzip.Reader) {
		err = errors.Join(err, gzr.Close())
	}(gzr)

	tr := tar.NewReader(gzr)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		name := filepath.Clean(hdr.Name)

		// Prevent path traversal.
		if strings.HasPrefix(name, "..") {
			return fmt.Errorf("invalid archive path %q", hdr.Name)
		}

		target := filepath.Join(dst, name)

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}

		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}

			f, err := os.OpenFile(
				target,
				os.O_CREATE|os.O_TRUNC|os.O_WRONLY,
				fs.FileMode(hdr.Mode),
			)
			if err != nil {
				return err
			}

			_, copyErr := io.Copy(f, tr)
			closeErr := f.Close()

			if copyErr != nil || closeErr != nil {
				return errors.Join(copyErr, closeErr)
			}
		}
	}
}
