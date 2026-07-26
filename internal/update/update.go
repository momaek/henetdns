// Package update implements the explicit self-update flow: query the latest
// GitHub release, compare versions, and replace the running binary with a
// downloaded release asset. Nothing here runs in the background — only the
// `version --check` and `upgrade` commands call into it.
package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/momaek/henetdns/internal/errs"
)

const Repo = "momaek/henetdns"

// binaryLimit caps extracted binary size as a safety net against a
// corrupt/malicious archive.
const binaryLimit = 200 << 20

// LatestTag returns the tag name (e.g. "v0.1.1") of the newest GitHub release.
func LatestTag(ctx context.Context, client *http.Client) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/"+Repo+"/releases/latest", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch latest release: %v: %w", err, errs.ErrRemote)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch latest release: unexpected status %s: %w", resp.Status, errs.ErrRemote)
	}
	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode latest release: %v: %w", err, errs.ErrRemote)
	}
	if payload.TagName == "" {
		return "", fmt.Errorf("latest release has no tag name: %w", errs.ErrRemote)
	}
	return payload.TagName, nil
}

// IsRelease reports whether v looks like a release version (vX.Y.Z or X.Y.Z),
// as opposed to "dev" or a pseudo-version.
func IsRelease(v string) bool {
	_, ok := parseVersion(v)
	return ok
}

// Compare orders two release versions numerically, ignoring a leading "v" and
// any pre-release suffix. Returns -1, 0, or 1. Non-release versions compare
// as older than any release.
func Compare(a, b string) int {
	av, aok := parseVersion(a)
	bv, bok := parseVersion(b)
	if !aok || !bok {
		switch {
		case aok == bok:
			return 0
		case bok:
			return -1
		default:
			return 1
		}
	}
	for i := 0; i < 3; i++ {
		if av[i] != bv[i] {
			if av[i] < bv[i] {
				return -1
			}
			return 1
		}
	}
	return 0
}

func parseVersion(v string) ([3]int, bool) {
	var out [3]int
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

// AssetName returns the goreleaser archive name for a tag and platform,
// e.g. henetdns_0.1.1_darwin_arm64.tar.gz (zip on windows).
func AssetName(tag, goos, goarch string) string {
	ext := "tar.gz"
	if goos == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("henetdns_%s_%s_%s.%s", strings.TrimPrefix(tag, "v"), goos, goarch, ext)
}

// Upgrade downloads the release asset for tag, verifies its checksum, and
// atomically replaces the current executable. Returns the path that was
// replaced.
func Upgrade(ctx context.Context, client *http.Client, tag string) (string, error) {
	asset := AssetName(tag, runtime.GOOS, runtime.GOARCH)
	base := "https://github.com/" + Repo + "/releases/download/" + tag + "/"

	archive, err := fetch(ctx, client, base+asset)
	if err != nil {
		return "", fmt.Errorf("download %s: %v: %w", asset, err, errs.ErrRemote)
	}
	sums, err := fetch(ctx, client, base+"checksums.txt")
	if err != nil {
		return "", fmt.Errorf("download checksums.txt: %v: %w", err, errs.ErrRemote)
	}
	if err := verifyChecksum(sums, asset, archive); err != nil {
		return "", err
	}

	bin, err := extractBinary(archive, asset)
	if err != nil {
		return "", err
	}

	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	if err := replaceBinary(exe, bin); err != nil {
		return "", err
	}
	return exe, nil
}

func fetch(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %s", resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, binaryLimit))
}

func verifyChecksum(sums []byte, asset string, archive []byte) error {
	for _, line := range strings.Split(string(sums), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || fields[1] != asset {
			continue
		}
		sum := sha256.Sum256(archive)
		if got := hex.EncodeToString(sum[:]); got != strings.ToLower(fields[0]) {
			return fmt.Errorf("checksum mismatch for %s: %w", asset, errs.ErrRemote)
		}
		return nil
	}
	return fmt.Errorf("no checksum entry for %s: %w", asset, errs.ErrRemote)
}

func extractBinary(archive []byte, asset string) ([]byte, error) {
	want := "henetdns"
	if strings.HasSuffix(asset, ".zip") {
		want += ".exe"
		return extractZip(archive, want)
	}
	return extractTarGz(archive, want)
}

func extractTarGz(archive []byte, want string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("open archive: %v: %w", err, errs.ErrRemote)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read archive: %v: %w", err, errs.ErrRemote)
		}
		if hdr.Typeflag == tar.TypeReg && filepath.Base(hdr.Name) == want {
			return io.ReadAll(io.LimitReader(tr, binaryLimit))
		}
	}
	return nil, fmt.Errorf("binary %q not found in archive: %w", want, errs.ErrRemote)
}

func extractZip(archive []byte, want string) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, fmt.Errorf("open archive: %v: %w", err, errs.ErrRemote)
	}
	for _, f := range zr.File {
		if filepath.Base(f.Name) != want {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("read archive: %v: %w", err, errs.ErrRemote)
		}
		defer rc.Close()
		return io.ReadAll(io.LimitReader(rc, binaryLimit))
	}
	return nil, fmt.Errorf("binary %q not found in archive: %w", want, errs.ErrRemote)
}

// replaceBinary writes bin next to exe and renames it into place, keeping the
// swap atomic on POSIX. Windows cannot rename over a running executable, so
// the old binary is moved aside first.
func replaceBinary(exe string, bin []byte) error {
	mode := os.FileMode(0o755)
	if info, err := os.Stat(exe); err == nil {
		mode = info.Mode().Perm()
	}

	tmp, err := os.CreateTemp(filepath.Dir(exe), ".henetdns-upgrade-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(bin); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, mode); err != nil {
		return err
	}

	if runtime.GOOS == "windows" {
		old := exe + ".old"
		_ = os.Remove(old)
		if err := os.Rename(exe, old); err != nil {
			return err
		}
		if err := os.Rename(tmpPath, exe); err != nil {
			_ = os.Rename(old, exe)
			return err
		}
		_ = os.Remove(old)
		return nil
	}
	return os.Rename(tmpPath, exe)
}
