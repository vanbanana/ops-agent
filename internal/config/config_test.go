package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnv_DoubleQuotes(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")

	content := `# OPS-Agent Configuration (auto-generated)
LLM_API_KEY="sk-test-abc123"
LLM_BASE_URL="https://api.deepseek.com"
LLM_MODEL="deepseek-v4-flash"
PORT="8080"
DB_PATH="./data/ops-agent.db"
JWT_SECRET="ops-deadbeef12345678"
ADMIN_PASSWORD="admin123"
`
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	os.Clearenv()
	LoadDotEnv(envFile)

	cases := []struct {
		key, want string
	}{
		{"PORT", "8080"},
		{"LLM_API_KEY", "sk-test-abc123"},
		{"LLM_BASE_URL", "https://api.deepseek.com"},
		{"LLM_MODEL", "deepseek-v4-flash"},
		{"DB_PATH", "./data/ops-agent.db"},
		{"JWT_SECRET", "ops-deadbeef12345678"},
		{"ADMIN_PASSWORD", "admin123"},
	}
	for _, c := range cases {
		got := os.Getenv(c.key)
		if got != c.want {
			t.Errorf("os.Getenv(%q) = %q, want %q", c.key, got, c.want)
		}
	}
}

func TestLoadDotEnv_SingleQuotes(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")

	content := `PORT='9090'
LLM_API_KEY='sk-single-quote'
`
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	os.Clearenv()
	LoadDotEnv(envFile)

	if v := os.Getenv("PORT"); v != "9090" {
		t.Errorf("PORT = %q, want %q", v, "9090")
	}
	if v := os.Getenv("LLM_API_KEY"); v != "sk-single-quote" {
		t.Errorf("LLM_API_KEY = %q, want %q", v, "sk-single-quote")
	}
}

func TestLoadDotEnv_NoQuotes(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")

	content := `PORT=3000
LLM_API_KEY=sk-no-quotes
`
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	os.Clearenv()
	LoadDotEnv(envFile)

	if v := os.Getenv("PORT"); v != "3000" {
		t.Errorf("PORT = %q, want %q", v, "3000")
	}
	if v := os.Getenv("LLM_API_KEY"); v != "sk-no-quotes" {
		t.Errorf("LLM_API_KEY = %q, want %q", v, "sk-no-quotes")
	}
}

func TestLoadDotEnv_MixedFormats(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")

	content := `PORT="8080"
LLM_API_KEY=sk-plain
LLM_MODEL='my-model'
# comment line

LLM_BASE_URL="https://example.com"
`
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	os.Clearenv()
	LoadDotEnv(envFile)

	if v := os.Getenv("PORT"); v != "8080" {
		t.Errorf("PORT = %q, want %q", v, "8080")
	}
	if v := os.Getenv("LLM_API_KEY"); v != "sk-plain" {
		t.Errorf("LLM_API_KEY = %q, want %q", v, "sk-plain")
	}
	if v := os.Getenv("LLM_MODEL"); v != "my-model" {
		t.Errorf("LLM_MODEL = %q, want %q", v, "my-model")
	}
	if v := os.Getenv("LLM_BASE_URL"); v != "https://example.com" {
		t.Errorf("LLM_BASE_URL = %q, want %q", v, "https://example.com")
	}
}

func TestLoadDotEnv_ExistingEnvNotOverwritten(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")

	content := `PORT="9999"
`
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	os.Clearenv()
	os.Setenv("PORT", "7777")
	LoadDotEnv(envFile)

	if v := os.Getenv("PORT"); v != "7777" {
		t.Errorf("PORT = %q, want %q (existing env should not be overwritten)", v, "7777")
	}
}

func TestLoadDotEnv_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")

	if err := os.WriteFile(envFile, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	os.Clearenv()
	LoadDotEnv(envFile)
}

func TestLoadDotEnv_NonexistentFile(t *testing.T) {
	os.Clearenv()
	LoadDotEnv("/nonexistent/path/.env")
}
