package main

import "testing"

func TestMaskToken(t *testing.T) {
	cases := []struct {
		name  string
		token string
		want  string
	}{
		{"empty", "", "(not set)"},
		{"short token fully masked", "abc", "****"},
		{"exactly eight chars masked", "12345678", "****"},
		{"nine chars reveals edges", "123456789", "1234****6789"},
		{"long token reveals edges", "abcdefghijklmnop", "abcd****mnop"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := maskToken(tc.token); got != tc.want {
				t.Fatalf("maskToken(%q) = %q, want %q", tc.token, got, tc.want)
			}
		})
	}
}
