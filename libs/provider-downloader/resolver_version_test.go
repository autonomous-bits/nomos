package downloader

import (
	"testing"
)

func TestFindMatchingAsset_WithVersion(t *testing.T) {
	tests := []struct {
		name       string
		assets     []githubAsset
		repo       string
		version    string
		targetOS   string
		targetArch string
		want       string
	}{
		{
			name: "exact match with version and extension",
			assets: []githubAsset{
				{Name: "nomos-provider-file-0.1.1-darwin-arm64.tar.gz"},
				{Name: "nomos-provider-file-0.1.1-linux-amd64.tar.gz"},
			},
			repo:       "nomos-provider-file",
			version:    "v0.1.1",
			targetOS:   "darwin",
			targetArch: "arm64",
			want:       "nomos-provider-file-0.1.1-darwin-arm64.tar.gz",
		},
		{
			name: "exact match with version no extension",
			assets: []githubAsset{
				{Name: "nomos-provider-file-0.1.0-darwin-arm64"},
				{Name: "nomos-provider-file-0.1.0-linux-amd64"},
			},
			repo:       "nomos-provider-file",
			version:    "v0.1.0",
			targetOS:   "darwin",
			targetArch: "arm64",
			want:       "nomos-provider-file-0.1.0-darwin-arm64",
		},
		{
			name: "substring match prefers version in filename",
			assets: []githubAsset{
				{Name: "provider-darwin-arm64.tar.gz"},
				{Name: "nomos-provider-file-1.2.3-darwin-arm64.tar.gz"},
			},
			repo:       "nomos-provider-file",
			version:    "v1.2.3",
			targetOS:   "darwin",
			targetArch: "arm64",
			want:       "nomos-provider-file-1.2.3-darwin-arm64.tar.gz",
		},
		{
			name: "legacy pattern still works when version not in filename",
			assets: []githubAsset{
				{Name: "nomos-provider-file-darwin-arm64"},
				{Name: "nomos-provider-file-linux-amd64"},
			},
			repo:       "nomos-provider-file",
			version:    "v1.0.0",
			targetOS:   "darwin",
			targetArch: "arm64",
			want:       "nomos-provider-file-darwin-arm64",
		},
		{
			name: "prefer exact pattern with version over legacy",
			assets: []githubAsset{
				{Name: "nomos-provider-file-darwin-arm64"},
				{Name: "nomos-provider-file-2.0.0-darwin-arm64"},
			},
			repo:       "nomos-provider-file",
			version:    "v2.0.0",
			targetOS:   "darwin",
			targetArch: "arm64",
			want:       "nomos-provider-file-2.0.0-darwin-arm64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{}
			got := client.findMatchingAsset(tt.assets, tt.repo, tt.version, tt.targetOS, tt.targetArch)
			if got != tt.want {
				t.Errorf("findMatchingAsset() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindMatchingAsset_ArchVariants(t *testing.T) {
	tests := []struct {
		name       string
		assets     []githubAsset
		repo       string
		version    string
		targetOS   string
		targetArch string
		want       string
	}{
		{
			name: "arm64 matches aarch64 in filename",
			assets: []githubAsset{
				{Name: "nomos-provider-file-1.0.0-linux-aarch64.tar.gz"},
			},
			repo:       "nomos-provider-file",
			version:    "v1.0.0",
			targetOS:   "linux",
			targetArch: "arm64",
			want:       "nomos-provider-file-1.0.0-linux-aarch64.tar.gz",
		},
		{
			name: "amd64 matches x86_64 in filename",
			assets: []githubAsset{
				{Name: "nomos-provider-file-1.0.0-linux-x86_64.tar.gz"},
			},
			repo:       "nomos-provider-file",
			version:    "v1.0.0",
			targetOS:   "linux",
			targetArch: "amd64",
			want:       "nomos-provider-file-1.0.0-linux-x86_64.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{}
			got := client.findMatchingAsset(tt.assets, tt.repo, tt.version, tt.targetOS, tt.targetArch)
			if got != tt.want {
				t.Errorf("findMatchingAsset() = %v, want %v", got, tt.want)
			}
		})
	}
}
