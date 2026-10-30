package config

import (
	"os"
	"testing"
)

func TestLoadMissingReturnsDefault(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg != Default() {
		t.Fatalf("got %+v, want defaults", cfg)
	}
}

func TestSaveLoad(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	want := Config{DataDir: "/tmp/rides", AutoSyncRides: true}
	if err := Save(want); err != nil {
		t.Fatal(err)
	}
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestLoadInvalid(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	path, _ := Path()
	if err := Save(Default()); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("data_dir = "), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err == nil {
		t.Fatal("expected error for invalid TOML")
	}
}
