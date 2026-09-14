package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	c := Default("/abs/MainShow.qlab5")
	if c.TargetPath != "/abs/MainShow.qlab5" {
		t.Errorf("TargetPath = %q", c.TargetPath)
	}
	if c.MinIntervalSeconds != 300 {
		t.Errorf("MinIntervalSeconds = %d", c.MinIntervalSeconds)
	}
	if c.DebounceMS != 300 {
		t.Errorf("DebounceMS = %d", c.DebounceMS)
	}
	if c.MaxSnapshots != 50 {
		t.Errorf("MaxSnapshots = %d", c.MaxSnapshots)
	}
}

func TestValidate(t *testing.T) {
	valid := Default("/abs/x")
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}

	cases := []struct {
		name   string
		mutate func(*Config)
	}{
		{"empty target", func(c *Config) { c.TargetPath = "" }},
		{"debounce zero", func(c *Config) { c.DebounceMS = 0 }},
		{"debounce negative", func(c *Config) { c.DebounceMS = -1 }},
		{"max snapshots zero", func(c *Config) { c.MaxSnapshots = 0 }},
		{"min interval negative", func(c *Config) { c.MinIntervalSeconds = -1 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := valid
			tc.mutate(&c)
			if err := c.Validate(); err == nil {
				t.Fatalf("expected error for %s", tc.name)
			}
		})
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	c := Default("/abs/MainShow.qlab5")
	c.MinIntervalSeconds = 60
	if err := c.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != c {
		t.Errorf("round-trip mismatch: got %+v want %+v", got, c)
	}
}

func TestLoadMalformed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected error for malformed config")
	}
}

func TestResolvePaths(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "MainShow.qlab5")
	p, err := ResolvePaths(target)
	if err != nil {
		t.Fatalf("ResolvePaths: %v", err)
	}
	if p.TargetAbs != target {
		t.Errorf("TargetAbs = %q want %q", p.TargetAbs, target)
	}
	if p.TargetDir != dir {
		t.Errorf("TargetDir = %q want %q", p.TargetDir, dir)
	}
	if p.Namespace != filepath.Join(dir, ".soute_MainShow.qlab5") {
		t.Errorf("Namespace = %q", p.Namespace)
	}
	if p.DataDir != filepath.Join(dir, ".soute_MainShow.qlab5", "data") {
		t.Errorf("DataDir = %q", p.DataDir)
	}
}

func TestResolvePathsSanitizesBasename(t *testing.T) {
	dir := t.TempDir()
	p, err := ResolvePaths(filepath.Join(dir, "my show(v2).qlab5"))
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, ".soute_my_show_v2_.qlab5")
	if p.Namespace != want {
		t.Errorf("Namespace = %q want %q", p.Namespace, want)
	}
}

func TestEnsure(t *testing.T) {
	dir := t.TempDir()
	p, err := ResolvePaths(filepath.Join(dir, "x.als"))
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Ensure(); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	for _, d := range []string{p.Namespace, p.DataDir} {
		if fi, err := os.Stat(d); err != nil || !fi.IsDir() {
			t.Errorf("expected directory %s (err=%v)", d, err)
		}
	}
}
