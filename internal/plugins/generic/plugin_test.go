package genericplugin

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestProcessEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single env var",
			input:    "Hello {{env:USER}}",
			expected: "Hello " + os.Getenv("USER"),
		},
		{
			name:     "multiple env vars",
			input:    "User: {{env:USER}}, Home: {{env:HOME}}",
			expected: "User: " + os.Getenv("USER") + ", Home: " + os.Getenv("HOME"),
		},
		{
			name:     "undefined env var",
			input:    "Value: {{env:NONEXISTENT_VAR_12345}}",
			expected: "Value: ",
		},
		{
			name:     "env var at end",
			input:    "Path is {{env:PATH}}",
			expected: "Path is " + os.Getenv("PATH"),
		},
		{
			name:     "env var at start",
			input:    "{{env:HOME}} is home",
			expected: os.Getenv("HOME") + " is home",
		},
		{
			name:     "no env vars",
			input:    "No placeholders here",
			expected: "No placeholders here",
		},
		{
			name:     "malformed placeholder",
			input:    "Hello {{env:USER",
			expected: "Hello {{env:USER",
		},
		{
			name:     "empty env var name",
			input:    "Value: {{env:}}",
			expected: "Value: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processEnvVars(tt.input)
			if result != tt.expected {
				t.Errorf("ProcessEnvVars(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestProcessFiles(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		content  string
		filename string
		expected string
	}{
		{
			name:     "single file",
			input:    "Hello {{file:%s}}",
			content:  "test",
			filename: "test.txt",
			expected: "Hello test",
		},
		{
			name:     "text before and after the placeholder",
			input:    "Prefix {{file:%s}} Suffix",
			content:  "test",
			filename: "test.txt",
			expected: "Prefix test Suffix",
		},
		{
			name:     "two placeholders in one string",
			input:    "{{file:%[1]s}} and {{file:%[1]s}}",
			content:  "test1\ntest2",
			filename: "test.txt",
			expected: "test1\ntest2 and test1\ntest2",
		},
		{
			name:     "md file as another example of text file",
			input:    "{{file:%s}}",
			content:  "# Hello\n## World\n\nThis is a test.",
			filename: "test.md",
			expected: "# Hello\n## World\n\nThis is a test.",
		},
		{
			name:     "csv file is converted to a markdown table",
			input:    "{{file:%s}}",
			content:  "name,age\nAlice,30\nBob,25\n",
			filename: "test.csv",
			expected: "| name | age |\n| --- | --- |\n| Alice | 30 |\n| Bob | 25 |\n",
		},
		{
			name:     "placeholder inside file content is not expanded",
			input:    "{{file:%s}}",
			content:  "see {{file:/does/not/exist.txt}} and {{env:HOME}}",
			filename: "test.txt",
			expected: "see {{file:/does/not/exist.txt}} and {{env:HOME}}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			filePath := filepath.Join(dir, tt.filename)
			input := fmt.Sprintf(tt.input, filePath)

			writeTestFile(t, filePath, tt.content)
			defer os.Remove(filePath)

			result := processFiles(input)
			if result != tt.expected {
				t.Errorf("processFiles(%q) = %q, want %q", input, result, tt.expected)
			}
		})
	}
}

func TestProcessFilesMultipleFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	first := filepath.Join(dir, "first.txt")
	second := filepath.Join(dir, "second.md")
	// t.TempDir() already removes dir; the explicit removal is a safeguard.
	writeTestFile(t, first, "one")
	defer os.Remove(first)
	writeTestFile(t, second, "# two")
	defer os.Remove(second)

	input := fmt.Sprintf("{{file:%s}} then {{file:%s}}", first, second)
	expected := "one then # two"

	result := processFiles(input)
	if result != expected {
		t.Errorf("processFiles(%q) = %q, want %q", input, result, expected)
	}
}

func TestProcessFilesUnchanged(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "no placeholders",
			input: "No placeholders here",
		},
		{
			name:  "missing closing braces",
			input: "Hello {{file:/tmp/some-file.txt",
		},
		{
			name:  "single closing brace",
			input: "Hello {{file:/tmp/some-file.txt}",
		},
		{
			name:  "other placeholder type",
			input: "Hello {{env:HOME}} and {{uuid}}",
		},
		{
			name:  "empty file name",
			input: "Hello {{file:}}",
		},
		{
			name:  "empty file name between other text",
			input: "before {{file:}} after",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := processFiles(tt.input)
			if result != tt.input {
				t.Errorf("processFiles(%q) = %q, want it unchanged", tt.input, result)
			}
		})
	}
}

// A placeholder whose file cannot be read (or is empty) stays in the text.
func TestProcessFilesKeepsPlaceholder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		create bool // whether the file is created before the call
	}{
		{
			name:   "file does not exist",
			create: false,
		},
		{
			name:   "file is empty",
			create: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "file.txt")
			if tt.create {
				writeTestFile(t, path, "")
				defer os.Remove(path)
			}

			input := fmt.Sprintf("before {{file:%s}} after", path)

			result := processFiles(input)
			if result != input {
				t.Errorf("processFiles(%q) = %q, want it unchanged", input, result)
			}
		})
	}
}

// An unreadable placeholder must not stop the following ones from
// being replaced.
func TestProcessFilesUnreadableThenReadable(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	missing := filepath.Join(dir, "missing.txt")
	existing := filepath.Join(dir, "existing.txt")
	writeTestFile(t, existing, "content")
	defer os.Remove(existing)

	input := fmt.Sprintf("{{file:%s}} and {{file:%s}}", missing, existing)
	expected := fmt.Sprintf("{{file:%s}} and content", missing)

	result := processFiles(input)
	if result != expected {
		t.Errorf("processFiles(%q) = %q, want %q", input, result, expected)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
