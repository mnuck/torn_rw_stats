package app

import (
	"os"
	"path/filepath"
	"testing"
)

func writeEnvFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadDotEnv(t *testing.T) {
	t.Run("sets missing vars", func(t *testing.T) {
		path := writeEnvFile(t, "DOTENV_TEST_KEY=hello\n")
		os.Unsetenv("DOTENV_TEST_KEY")
		defer os.Unsetenv("DOTENV_TEST_KEY")

		if err := LoadDotEnv(path); err != nil {
			t.Fatal(err)
		}
		if got := os.Getenv("DOTENV_TEST_KEY"); got != "hello" {
			t.Errorf("got %q, want %q", got, "hello")
		}
	})

	t.Run("does not overwrite existing vars", func(t *testing.T) {
		path := writeEnvFile(t, "DOTENV_TEST_EXISTING=file_value\n")
		os.Setenv("DOTENV_TEST_EXISTING", "original")
		defer os.Unsetenv("DOTENV_TEST_EXISTING")

		if err := LoadDotEnv(path); err != nil {
			t.Fatal(err)
		}
		if got := os.Getenv("DOTENV_TEST_EXISTING"); got != "original" {
			t.Errorf("got %q, want %q", got, "original")
		}
	})

	t.Run("skips comments and blank lines", func(t *testing.T) {
		path := writeEnvFile(t, "# this is a comment\n\nDOTENV_TEST_COMMENTED=set\n")
		os.Unsetenv("DOTENV_TEST_COMMENTED")
		defer os.Unsetenv("DOTENV_TEST_COMMENTED")

		if err := LoadDotEnv(path); err != nil {
			t.Fatal(err)
		}
		if got := os.Getenv("DOTENV_TEST_COMMENTED"); got != "set" {
			t.Errorf("got %q, want %q", got, "set")
		}
	})

	t.Run("strips double quotes", func(t *testing.T) {
		path := writeEnvFile(t, `DOTENV_TEST_QUOTED="my value"`+"\n")
		os.Unsetenv("DOTENV_TEST_QUOTED")
		defer os.Unsetenv("DOTENV_TEST_QUOTED")

		if err := LoadDotEnv(path); err != nil {
			t.Fatal(err)
		}
		if got := os.Getenv("DOTENV_TEST_QUOTED"); got != "my value" {
			t.Errorf("got %q, want %q", got, "my value")
		}
	})

	t.Run("strips single quotes", func(t *testing.T) {
		path := writeEnvFile(t, "DOTENV_TEST_SQUOTED='my value'\n")
		os.Unsetenv("DOTENV_TEST_SQUOTED")
		defer os.Unsetenv("DOTENV_TEST_SQUOTED")

		if err := LoadDotEnv(path); err != nil {
			t.Fatal(err)
		}
		if got := os.Getenv("DOTENV_TEST_SQUOTED"); got != "my value" {
			t.Errorf("got %q, want %q", got, "my value")
		}
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		if err := LoadDotEnv("/nonexistent/.env"); err == nil {
			t.Error("expected error for missing file, got nil")
		}
	})

	t.Run("value containing equals sign", func(t *testing.T) {
		path := writeEnvFile(t, "DOTENV_TEST_EQ=a=b=c\n")
		os.Unsetenv("DOTENV_TEST_EQ")
		defer os.Unsetenv("DOTENV_TEST_EQ")

		if err := LoadDotEnv(path); err != nil {
			t.Fatal(err)
		}
		if got := os.Getenv("DOTENV_TEST_EQ"); got != "a=b=c" {
			t.Errorf("got %q, want %q", got, "a=b=c")
		}
	})
}
