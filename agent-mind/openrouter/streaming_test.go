package openrouter

import (
	"testing"
)

func TestParseStreamingJson(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantKeys int
	}{
		{"empty", "", 0},
		{"whitespace", "  \n  ", 0},
		{"valid object", `{"a":1,"b":"x"}`, 2},
		{"partial JSON", `{"a":1`, 0},
		{"invalid", `not json`, 0},
		{"empty object", `{}`, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseStreamingJson(tt.input)
			if got == nil {
				t.Fatal("parseStreamingJson must not return nil")
			}
			if len(got) != tt.wantKeys {
				t.Errorf("parseStreamingJson(%q) keys = %d, want %d", tt.input, len(got), tt.wantKeys)
			}
		})
	}
}
