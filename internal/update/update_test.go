package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"v0.1.1", "v0.1.1", 0},
		{"0.1.1", "v0.1.1", 0},
		{"v0.1.1", "v0.1.2", -1},
		{"v0.1.2", "v0.1.1", 1},
		{"v0.1.1", "v0.2.0", -1},
		{"v0.9.9", "v1.0.0", -1},
		{"v1.0", "v1.0.0", 0},
		{"v1.2.3-rc1", "v1.2.3", 0},
		{"dev", "v0.1.1", -1},
		{"v0.1.1", "dev", 1},
		{"dev", "dev", 0},
	}
	for _, c := range cases {
		if got := Compare(c.a, c.b); got != c.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestIsRelease(t *testing.T) {
	for v, want := range map[string]bool{
		"v0.1.1":  true,
		"0.1.1":   true,
		"v1.2":    true,
		"dev":     false,
		"(devel)": false,
		"":        false,
		"v0.1.1-0.20260726150405-abcdef123456": true,
	} {
		if got := IsRelease(v); got != want {
			t.Errorf("IsRelease(%q) = %v, want %v", v, got, want)
		}
	}
}

func TestAssetName(t *testing.T) {
	if got, want := AssetName("v0.1.1", "darwin", "arm64"), "henetdns_0.1.1_darwin_arm64.tar.gz"; got != want {
		t.Errorf("AssetName = %q, want %q", got, want)
	}
	if got, want := AssetName("v0.1.1", "windows", "amd64"), "henetdns_0.1.1_windows_amd64.zip"; got != want {
		t.Errorf("AssetName = %q, want %q", got, want)
	}
}

func TestVerifyChecksum(t *testing.T) {
	archive := []byte("archive-bytes")
	sum := sha256.Sum256(archive)
	asset := "henetdns_0.1.1_darwin_arm64.tar.gz"
	sums := []byte("deadbeef  other.tar.gz\n" + hex.EncodeToString(sum[:]) + "  " + asset + "\n")

	if err := verifyChecksum(sums, asset, archive); err != nil {
		t.Errorf("valid checksum rejected: %v", err)
	}
	if err := verifyChecksum(sums, asset, []byte("tampered")); err == nil {
		t.Error("tampered archive accepted")
	}
	if err := verifyChecksum(sums, "missing.tar.gz", archive); err == nil {
		t.Error("missing checksum entry accepted")
	}
}

func TestExtractTarGz(t *testing.T) {
	content := []byte("fake-binary")
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range map[string][]byte{"README.md": []byte("docs"), "henetdns": content} {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}

	got, err := extractBinary(buf.Bytes(), "henetdns_0.1.1_darwin_arm64.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("extracted %q, want %q", got, content)
	}
}

func TestExtractZip(t *testing.T) {
	content := []byte("fake-binary")
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range map[string][]byte{"README.md": []byte("docs"), "henetdns.exe": content} {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	got, err := extractBinary(buf.Bytes(), "henetdns_0.1.1_windows_amd64.zip")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("extracted %q, want %q", got, content)
	}
}
