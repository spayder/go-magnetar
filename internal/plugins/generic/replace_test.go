package genericplugin

import "testing"

// maxReplCalls guards against an endless loop in replaceAllPatterns:
// once it is exceeded the replacement function returns text without
// the prefix, which makes the loop stop and the test fail.
const maxReplCalls = 20

func TestReplaceAllPatterns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		repl      func(string) string
		expected  string
		wantCalls int
	}{
		{
			name:  "replacement is applied",
			input: "a {{file:x}} b",
			repl: func(string) string {
				return "X"
			},
			expected:  "a X b",
			wantCalls: 1,
		},
		{
			name:  "unchanged placeholder is not processed again",
			input: "a {{file:x}} b",
			repl: func(match string) string {
				return match
			},
			expected:  "a {{file:x}} b",
			wantCalls: 1,
		},
		{
			name:  "several unchanged placeholders",
			input: "{{file:a}} and {{file:b}}",
			repl: func(match string) string {
				return match
			},
			expected:  "{{file:a}} and {{file:b}}",
			wantCalls: 2,
		},
		{
			name:  "unchanged placeholder followed by replaced one",
			input: "{{file:bad}} and {{file:good}}",
			repl: func(match string) string {
				if match == "{{file:good}}" {
					return "OK"
				}
				return match
			},
			expected:  "{{file:bad}} and OK",
			wantCalls: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			calls := 0
			repl := func(match string) string {
				calls++
				if calls > maxReplCalls {
					return ""
				}
				return tt.repl(match)
			}

			result := replaceAllPatterns(tt.input, "{{file:", "}}", repl)

			if result != tt.expected {
				t.Errorf("replaceAllPatterns(%q) = %q, want %q", tt.input, result, tt.expected)
			}
			if calls != tt.wantCalls {
				t.Errorf("replaceAllPatterns(%q) called repl %d times, want %d", tt.input, calls, tt.wantCalls)
			}
		})
	}
}
