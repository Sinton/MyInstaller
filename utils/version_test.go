package utils

import (
	"testing"
)

func TestParseVersion(t *testing.T) {
	vc := NewVersionComparator()

	tests := []struct {
		name    string
		output  string
		want    string
		wantErr bool
	}{
		{
			name:    "simple version",
			output:  "1.2.3",
			want:    "1.2.3",
			wantErr: false,
		},
		{
			name:    "version with v prefix",
			output:  "v1.2.3",
			want:    "1.2.3",
			wantErr: false,
		},
		{
			name:    "npm output format",
			output:  "pnpm@8.15.4",
			want:    "8.15.4",
			wantErr: false,
		},
		{
			name:    "scoped package",
			output:  "@typescript-eslint/parser@6.21.0",
			want:    "6.21.0",
			wantErr: false,
		},
		{
			name:    "version with whitespace",
			output:  "  1.2.3  \n",
			want:    "1.2.3",
			wantErr: false,
		},
		{
			name:    "pre-release version",
			output:  "1.2.3-alpha.1",
			want:    "1.2.3-alpha.1",
			wantErr: false,
		},
		{
			name:    "version with build metadata",
			output:  "1.2.3+build.123",
			want:    "1.2.3+build.123",
			wantErr: false,
		},
		{
			name:    "pre-release with build metadata",
			output:  "1.2.3-beta.2+build.456",
			want:    "1.2.3-beta.2+build.456",
			wantErr: false,
		},
		{
			name:    "empty output",
			output:  "",
			want:    "",
			wantErr: true,
		},
		{
			name:    "no version in output",
			output:  "command not found",
			want:    "",
			wantErr: true,
		},
		{
			name:    "invalid version format",
			output:  "1.2",
			want:    "",
			wantErr: true,
		},
		// Additional edge cases for comprehensive coverage
		{
			name:    "npm list output with tree structure",
			output:  "└── typescript@5.3.3",
			want:    "5.3.3",
			wantErr: false,
		},
		{
			name:    "npm list output with path",
			output:  "/usr/local/lib/node_modules/pnpm@8.15.4",
			want:    "8.15.4",
			wantErr: false,
		},
		{
			name:    "version with multiple @ symbols",
			output:  "@babel/core@7.23.9",
			want:    "7.23.9",
			wantErr: false,
		},
		{
			name:    "version with extra text before",
			output:  "latest: 10.2.4",
			want:    "10.2.4",
			wantErr: false,
		},
		{
			name:    "version with extra text after",
			output:  "1.2.3 (latest)",
			want:    "1.2.3",
			wantErr: false,
		},
		{
			name:    "pre-release with numeric suffix",
			output:  "2.0.0-rc.1",
			want:    "2.0.0-rc.1",
			wantErr: false,
		},
		{
			name:    "pre-release with multiple dots",
			output:  "1.0.0-alpha.beta.1",
			want:    "1.0.0-alpha.beta.1",
			wantErr: false,
		},
		{
			name:    "version with tabs and newlines",
			output:  "\t\n1.2.3\n\t",
			want:    "1.2.3",
			wantErr: false,
		},
		{
			name:    "large version numbers",
			output:  "100.200.300",
			want:    "100.200.300",
			wantErr: false,
		},
		{
			name:    "version zero",
			output:  "0.0.0",
			want:    "0.0.0",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := vc.ParseVersion(tt.output)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompareVersions(t *testing.T) {
	vc := NewVersionComparator()

	tests := []struct {
		name    string
		v1      string
		v2      string
		want    int
		wantErr bool
	}{
		{
			name:    "v1 greater than v2",
			v1:      "2.0.0",
			v2:      "1.0.0",
			want:    1,
			wantErr: false,
		},
		{
			name:    "v1 less than v2",
			v1:      "1.0.0",
			v2:      "2.0.0",
			want:    -1,
			wantErr: false,
		},
		{
			name:    "v1 equals v2",
			v1:      "1.2.3",
			v2:      "1.2.3",
			want:    0,
			wantErr: false,
		},
		{
			name:    "minor version difference",
			v1:      "1.5.0",
			v2:      "1.4.0",
			want:    1,
			wantErr: false,
		},
		{
			name:    "patch version difference",
			v1:      "1.2.4",
			v2:      "1.2.3",
			want:    1,
			wantErr: false,
		},
		{
			name:    "pre-release versions",
			v1:      "1.0.0-alpha",
			v2:      "1.0.0-beta",
			want:    -1,
			wantErr: false,
		},
		{
			name:    "release vs pre-release",
			v1:      "1.0.0",
			v2:      "1.0.0-alpha",
			want:    1,
			wantErr: false,
		},
		{
			name:    "invalid v1",
			v1:      "invalid",
			v2:      "1.0.0",
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid v2",
			v1:      "1.0.0",
			v2:      "invalid",
			want:    0,
			wantErr: true,
		},
		// Additional edge cases for comprehensive coverage
		{
			name:    "major version difference with same minor and patch",
			v1:      "3.2.1",
			v2:      "2.2.1",
			want:    1,
			wantErr: false,
		},
		{
			name:    "large version numbers",
			v1:      "100.200.300",
			v2:      "100.200.299",
			want:    1,
			wantErr: false,
		},
		{
			name:    "version zero comparison",
			v1:      "0.0.0",
			v2:      "0.0.1",
			want:    -1,
			wantErr: false,
		},
		{
			name:    "pre-release with numeric suffix",
			v1:      "1.0.0-rc.2",
			v2:      "1.0.0-rc.1",
			want:    1,
			wantErr: false,
		},
		{
			name:    "pre-release vs pre-release with different identifiers",
			v1:      "1.0.0-alpha.1",
			v2:      "1.0.0-alpha.2",
			want:    -1,
			wantErr: false,
		},
		{
			name:    "build metadata should be ignored in comparison",
			v1:      "1.0.0+build.1",
			v2:      "1.0.0+build.2",
			want:    0,
			wantErr: false,
		},
		{
			name:    "pre-release with build metadata",
			v1:      "1.0.0-alpha+build.1",
			v2:      "1.0.0-alpha+build.2",
			want:    0,
			wantErr: false,
		},
		{
			name:    "different pre-release stages",
			v1:      "1.0.0-rc.1",
			v2:      "1.0.0-beta.1",
			want:    1,
			wantErr: false,
		},
		{
			name:    "pre-release with multiple identifiers",
			v1:      "1.0.0-alpha.beta.1",
			v2:      "1.0.0-alpha.beta.2",
			want:    -1,
			wantErr: false,
		},
		{
			name:    "same version with and without build metadata",
			v1:      "1.0.0",
			v2:      "1.0.0+build.123",
			want:    0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := vc.CompareVersions(tt.v1, tt.v2)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompareVersions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CompareVersions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsNewerVersion(t *testing.T) {
	vc := NewVersionComparator()

	tests := []struct {
		name string
		v1   string
		v2   string
		want bool
	}{
		{
			name: "v1 is newer",
			v1:   "2.0.0",
			v2:   "1.0.0",
			want: true,
		},
		{
			name: "v1 is older",
			v1:   "1.0.0",
			v2:   "2.0.0",
			want: false,
		},
		{
			name: "versions are equal",
			v1:   "1.2.3",
			v2:   "1.2.3",
			want: false,
		},
		{
			name: "minor version newer",
			v1:   "1.5.0",
			v2:   "1.4.0",
			want: true,
		},
		{
			name: "patch version newer",
			v1:   "1.2.4",
			v2:   "1.2.3",
			want: true,
		},
		{
			name: "release is newer than pre-release",
			v1:   "1.0.0",
			v2:   "1.0.0-alpha",
			want: true,
		},
		{
			name: "invalid v1 returns false",
			v1:   "invalid",
			v2:   "1.0.0",
			want: false,
		},
		{
			name: "invalid v2 returns false",
			v1:   "1.0.0",
			v2:   "invalid",
			want: false,
		},
		// Additional edge cases for comprehensive coverage
		{
			name: "major version bump",
			v1:   "3.0.0",
			v2:   "2.9.9",
			want: true,
		},
		{
			name: "minor version bump with lower patch",
			v1:   "1.5.0",
			v2:   "1.4.9",
			want: true,
		},
		{
			name: "pre-release newer than older pre-release",
			v1:   "1.0.0-beta",
			v2:   "1.0.0-alpha",
			want: true,
		},
		{
			name: "rc newer than beta",
			v1:   "1.0.0-rc.1",
			v2:   "1.0.0-beta.1",
			want: true,
		},
		{
			name: "pre-release with higher number",
			v1:   "1.0.0-alpha.2",
			v2:   "1.0.0-alpha.1",
			want: true,
		},
		{
			name: "build metadata ignored - equal versions",
			v1:   "1.0.0+build.2",
			v2:   "1.0.0+build.1",
			want: false,
		},
		{
			name: "version zero is not newer than zero",
			v1:   "0.0.0",
			v2:   "0.0.0",
			want: false,
		},
		{
			name: "large version numbers comparison",
			v1:   "100.200.301",
			v2:   "100.200.300",
			want: true,
		},
		{
			name: "both invalid returns false",
			v1:   "invalid1",
			v2:   "invalid2",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := vc.IsNewerVersion(tt.v1, tt.v2)
			if got != tt.want {
				t.Errorf("IsNewerVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}
