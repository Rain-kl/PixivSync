// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package updater

import (
	"os"
	"runtime"
	"testing"
	"time"
)

func TestParseRepository(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "short form", input: "Rain-kl/Wavelet", want: "Rain-kl/Wavelet"},
		{name: "GitHub URL", input: "https://github.com/Rain-kl/Wavelet.git", want: "Rain-kl/Wavelet"},
		{name: "unsupported host", input: "https://example.com/Rain-kl/Wavelet", wantErr: true},
		{name: "missing owner", input: "Wavelet", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRepository(tt.input)
			if gotErr := err != nil; gotErr != tt.wantErr {
				t.Errorf("parseRepository(%q) error = %v, want error presence = %t", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("parseRepository(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSelectLatestRelease(t *testing.T) {
	assetNameV1 := expectedAssetName("v1.0.0")
	assetNameV2 := expectedAssetName("v2.0.0")
	releases := []githubRelease{
		{
			TagName:   "v1.0.0",
			Published: time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC),
			Assets: []releaseAsset{{
				Name:               assetNameV1,
				BrowserDownloadURL: "https://example.com/v1",
				State:              "uploaded",
			}},
		},
		{
			TagName:   "v2.0.0",
			Published: time.Date(2026, time.June, 2, 0, 0, 0, 0, time.UTC),
			Assets: []releaseAsset{{
				Name:               assetNameV2,
				BrowserDownloadURL: "https://example.com/v2",
				State:              "uploaded",
			}},
		},
		{
			TagName: "v3.0.0",
			Assets: []releaseAsset{{
				Name:               "wavelet_v3.0.0_other_platform.tar.gz",
				BrowserDownloadURL: "https://example.com/v3",
				State:              "uploaded",
			}},
		},
	}

	release, asset, err := selectLatestRelease("Rain-kl/Wavelet", releases)
	if err != nil {
		t.Fatalf("selectLatestRelease() error = %v", err)
	}
	if release.TagName != "v2.0.0" {
		t.Errorf("selectLatestRelease() tag = %q, want %q", release.TagName, "v2.0.0")
	}
	if asset.Name != assetNameV2 {
		t.Errorf("selectLatestRelease() asset = %q, want %q", asset.Name, assetNameV2)
	}
}

func TestSelectLatestReleaseWithCustomRepo(t *testing.T) {
	extension := "tar.gz"
	if runtime.GOOS == "windows" {
		extension = "zip"
	}
	releases := []githubRelease{
		{
			TagName:   "v1.0.0",
			Published: time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC),
			Assets: []releaseAsset{{
				Name:               "wavelet_v1.0.0_" + runtime.GOOS + "_" + runtime.GOARCH + "." + extension,
				BrowserDownloadURL: "https://example.com/v1",
				State:              "uploaded",
			}},
		},
		{
			TagName:   "v2.0.0",
			Published: time.Date(2026, time.June, 2, 0, 0, 0, 0, time.UTC),
			Assets: []releaseAsset{{
				Name:               "PixezSync_v2.0.0_" + runtime.GOOS + "_" + runtime.GOARCH + "." + extension,
				BrowserDownloadURL: "https://example.com/v2",
				State:              "uploaded",
			}},
		},
	}

	release, asset, err := selectLatestRelease("Rain-kl/PixezSync", releases)
	if err != nil {
		t.Fatalf("selectLatestRelease() error = %v", err)
	}
	if release.TagName != "v2.0.0" {
		t.Errorf("selectLatestRelease() tag = %q, want %q", release.TagName, "v2.0.0")
	}
	expectedName := "PixezSync_v2.0.0_" + runtime.GOOS + "_" + runtime.GOARCH + "." + extension
	if asset.Name != expectedName {
		t.Errorf("selectLatestRelease() asset = %q, want %q", asset.Name, expectedName)
	}
}

func TestExpectedAssetName(t *testing.T) {
	extension := "tar.gz"
	if runtime.GOOS == "windows" {
		extension = "zip"
	}
	want := "wavelet_v1.2.3_" + runtime.GOOS + "_" + runtime.GOARCH + "." + extension
	if got := expectedAssetName("v1.2.3"); got != want {
		t.Errorf("expectedAssetName(%q) = %q, want %q", "v1.2.3", got, want)
	}
}

func TestGetCandidateBinaryNames(t *testing.T) {
	var execPath string
	if runtime.GOOS == "windows" {
		execPath = `C:\bin\PixezSync.exe`
	} else {
		execPath = "/usr/local/bin/PixezSync"
	}

	candidates := getCandidateBinaryNames(execPath, "Rain-kl/PixezSync")
	expected := []string{"PixezSync", "wavelet"}
	if runtime.GOOS == "windows" {
		expected = []string{"PixezSync.exe", "wavelet.exe"}
	}

	if len(candidates) != len(expected) {
		t.Fatalf("getCandidateBinaryNames() returned length %d, want %d (got %v)", len(candidates), len(expected), candidates)
	}

	for i, exp := range expected {
		if candidates[i] != exp {
			t.Errorf("getCandidateBinaryNames()[%d] = %q, want %q", i, candidates[i], exp)
		}
	}
}

func TestMatchBinaryName(t *testing.T) {
	candidates := []string{"PixezSync", "wavelet"}
	if runtime.GOOS == "windows" {
		candidates = []string{"PixezSync.exe", "wavelet.exe"}
	}

	tests := []struct {
		name      string
		input     string
		wantMatch bool
	}{
		{
			name:      "exact match repo name",
			input:     "PixezSync",
			wantMatch: runtime.GOOS != "windows",
		},
		{
			name:      "exact match default name",
			input:     "wavelet",
			wantMatch: runtime.GOOS != "windows",
		},
		{
			name:      "exact match windows repo name",
			input:     "PixezSync.exe",
			wantMatch: runtime.GOOS == "windows",
		},
		{
			name:      "case insensitive match windows",
			input:     "pixezsync.exe",
			wantMatch: runtime.GOOS == "windows",
		},
		{
			name:      "unmatched name",
			input:     "other_binary",
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchBinaryName(tt.input, candidates)
			if got != tt.wantMatch {
				t.Errorf("matchBinaryName(%q) = %t, want %t", tt.input, got, tt.wantMatch)
			}
		})
	}
}

func TestIsLikelyBinary(t *testing.T) {
	tests := []struct {
		name      string
		filename  string
		isDir     bool
		mode      uint32
		wantMatch bool
	}{
		{
			name:      "markdown file",
			filename:  "README.md",
			isDir:     false,
			mode:      0644,
			wantMatch: false,
		},
		{
			name:      "license file",
			filename:  "LICENSE",
			isDir:     false,
			mode:      0644,
			wantMatch: false,
		},
		{
			name:      "directory",
			filename:  "bin",
			isDir:     true,
			mode:      0755,
			wantMatch: false,
		},
		{
			name:      "windows binary",
			filename:  "PixezSync.exe",
			isDir:     false,
			mode:      0644,
			wantMatch: runtime.GOOS == "windows",
		},
		{
			name:      "unix binary with no extension",
			filename:  "PixezSync",
			isDir:     false,
			mode:      0755,
			wantMatch: runtime.GOOS != "windows",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isLikelyBinary(tt.filename, tt.isDir, os.FileMode(tt.mode))
			if got != tt.wantMatch {
				t.Errorf("isLikelyBinary(%q, %t, %o) = %t, want %t", tt.filename, tt.isDir, tt.mode, got, tt.wantMatch)
			}
		})
	}
}
