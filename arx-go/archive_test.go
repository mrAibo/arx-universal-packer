package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompressExtractRoundtrip(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "data")
	os.Mkdir(src, 0o755)
	os.WriteFile(filepath.Join(src, "a.txt"), []byte("hello"), 0o644)
	os.WriteFile(filepath.Join(src, "b.log"), []byte("log"), 0o644)

	for _, fmt := range []string{"tar", "tar.gz", "tar.bz2", "tar.xz", "tar.zst", "zip", "7z"} {
		out := filepath.Join(dir, "out_"+fmt)
		os.Mkdir(out, 0o755)
		r := compress(fmt, "bk", src, out, 3)
		if r.Err != nil {
			t.Fatalf("%s compress: %v", fmt, r.Err)
		}
		arc := filepath.Join(out, "bk."+fmt)
		if _, err := os.Stat(arc); err != nil {
			t.Fatalf("%s archive missing", fmt)
		}
		ext := filepath.Join(out, "ext")
		os.Mkdir(ext, 0o755)
		if re := extract(arc, ext); re.Err != nil {
			t.Fatalf("%s extract: %v", fmt, re.Err)
		}
		if _, err := os.Stat(filepath.Join(ext, "data", "a.txt")); err != nil {
			t.Fatalf("%s roundtrip content lost", fmt)
		}
	}
}

func TestDetectFormat(t *testing.T) {
	cases := map[string]string{
		"a.tar.gz":  "tar.gz",
		"a.tgz":     "tar.gz",
		"a.zip":     "zip",
		"a.7z":      "7z",
		"a.tar.zst": "tar.zst",
		"a.unknown": "unknown",
	}
	for in, want := range cases {
		if got := DetectFormat(in); got != want {
			t.Errorf("DetectFormat(%q)=%q want %q", in, got, want)
		}
	}
}

func TestConvert(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "d")
	os.Mkdir(src, 0o755)
	os.WriteFile(filepath.Join(src, "x.txt"), []byte("x"), 0o644)
	r := compress("tar.gz", "c1", src, dir, 3)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
	dest := filepath.Join(dir, "c1.tar.zst")
	if rc := convert(filepath.Join(dir, "c1.tar.gz"), dest); rc.Err != nil {
		t.Fatal(rc.Err)
	}
	if _, err := os.Stat(dest); err != nil {
		t.Fatal("converted file missing")
	}
}
