package main

import "testing"

func TestParseVersionToken(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{
			name:   "release version",
			output: "timbers version v0.22.3 (e18819e, 2026-05-27T16:47:03Z)",
			want:   "v0.22.3",
		},
		{
			name:   "dev build",
			output: "timbers version dev",
			want:   "dev",
		},
		{
			name:   "describe-style dev token",
			output: "timbers version v0.22.2-13-ge18819e (e18819e, 2026-05-27T16:47:03Z)",
			want:   "v0.22.2-13-ge18819e",
		},
		{
			name:   "trailing newline",
			output: "timbers version v1.0.0\n",
			want:   "v1.0.0",
		},
		{
			name:   "unparseable output",
			output: "garbage with no marker",
			want:   "?",
		},
		{
			name:   "empty output",
			output: "",
			want:   "?",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseVersionToken(tt.output); got != tt.want {
				t.Errorf("parseVersionToken(%q) = %q, want %q", tt.output, got, tt.want)
			}
		})
	}
}
