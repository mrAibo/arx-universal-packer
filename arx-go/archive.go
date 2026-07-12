package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Result is what the backend returns to the UI.
type Result struct {
	Output string
	Err    error
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// runIn runs a command with the working directory changed to dir.
func runIn(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// runCapture runs a command and returns its stdout (used for list/verify text).
func runCapture(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	return string(out), err
}

// DetectFormat infers the archive format from the file extension.
func DetectFormat(path string) string {
	switch {
	case strings.HasSuffix(path, ".tar.gz"), strings.HasSuffix(path, ".tgz"):
		return "tar.gz"
	case strings.HasSuffix(path, ".tar.bz2"), strings.HasSuffix(path, ".tbz2"):
		return "tar.bz2"
	case strings.HasSuffix(path, ".tar.xz"), strings.HasSuffix(path, ".txz"):
		return "tar.xz"
	case strings.HasSuffix(path, ".tar.zst"):
		return "tar.zst"
	case strings.HasSuffix(path, ".tar"):
		return "tar"
	case strings.HasSuffix(path, ".zip"):
		return "zip"
	case strings.HasSuffix(path, ".7z"):
		return "7z"
	default:
		return "unknown"
	}
}

// compress builds and runs the archive command. source may be a file or dir.
func compress(format, name, source, targetDir string, level int) Result {
	src, err := filepath.Abs(source)
	if err != nil {
		return Result{Err: err}
	}
	if _, err := os.Stat(src); err != nil {
		return Result{Err: fmt.Errorf("file not found: %s", source)}
	}
	out := filepath.Join(targetDir, name+"."+format)
	// ponytail: remove stale output so we never silently overwrite a good archive
	if _, err := os.Stat(out); err == nil {
		os.Remove(out)
	}
	var cmdErr error
	switch format {
	case "tar":
		cmdErr = run("tar", "-cf", out, "-C", filepath.Dir(src), filepath.Base(src))
	case "tar.gz":
		l := fmt.Sprintf("-%d", level)
		comp := "gzip"
		if _, e := exec.LookPath("pigz"); e == nil {
			comp = "pigz"
		}
		cmdErr = pipeTar(out, src, comp, l)
	case "tar.bz2":
		l := fmt.Sprintf("-%d", level)
		comp := "bzip2"
		if _, e := exec.LookPath("pbzip2"); e == nil {
			comp = "pbzip2"
		}
		cmdErr = pipeTar(out, src, comp, l)
	case "tar.xz":
		os.Setenv("XZ_OPT", fmt.Sprintf("--threads=0 -%d", level))
		cmdErr = run("tar", "-cJf", out, "-C", filepath.Dir(src), filepath.Base(src))
	case "tar.zst":
		if _, e := exec.LookPath("zstd"); e == nil {
			cmdErr = pipeTar(out, src, "zstd", fmt.Sprintf("-%d", level))
		} else {
			cmdErr = run("tar", "--zstd", "-cf", out, "-C", filepath.Dir(src), filepath.Base(src))
		}
	case "zip":
		// ponytail: zip has no -C; cd to parent so archive stores relative paths
		l := fmt.Sprintf("-%d", level)
		parent := filepath.Dir(src)
		base := filepath.Base(src)
		cmdErr = runIn(parent, "zip", "-r", l, out, base)
	case "7z":
		// ponytail: 7z also stores absolute paths unless run from parent
		cmdErr = runIn(filepath.Dir(src), "7z", "a", fmt.Sprintf("-mx=%d", level), out, filepath.Base(src))
	default:
		return Result{Err: fmt.Errorf("unsupported format: %s", format)}
	}
	if cmdErr != nil {
		return Result{Err: cmdErr}
	}
	return Result{Output: "Archive created: " + out}
}

// pipeTar streams tar through an external compressor so multi-thread tools work.
func pipeTar(out, src, compressor, level string) error {
	tarc := exec.Command("tar", "-c", "-C", filepath.Dir(src), filepath.Base(src))
	comp := exec.Command(compressor, level, "-c")
	pr, pw := io.Pipe()
	tarc.Stdout = pw
	comp.Stdin = pr
	outF, err := os.Create(out)
	if err != nil {
		return err
	}
	defer outF.Close()
	comp.Stdout = outF
	if err := tarc.Start(); err != nil {
		return err
	}
	if err := comp.Start(); err != nil {
		return err
	}
	go func() { tarc.Wait(); pw.Close() }()
	return comp.Wait()
}

// extract unpacks an archive into targetDir (created if missing).
func extract(path, targetDir string) Result {
	src, err := filepath.Abs(path)
	if err != nil {
		return Result{Err: err}
	}
	if _, err := os.Stat(src); err != nil {
		return Result{Err: fmt.Errorf("file not found: %s", path)}
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return Result{Err: err}
	}
	format := DetectFormat(src)
	var cmdErr error
	switch f := format; f {
	case "tar":
		cmdErr = run("tar", "-x", "-f", src, "-C", targetDir)
	case "tar.gz", "tar.bz2", "tar.xz", "tar.zst":
		flag := ""
		switch f {
		case "tar.gz":
			flag = "-z"
		case "tar.bz2":
			flag = "-j"
		case "tar.xz":
			flag = "-J"
		case "tar.zst":
			flag = "--zstd"
		}
		cmdErr = run("tar", "-x", flag, "-f", src, "-C", targetDir)
	case "zip":
		cmdErr = run("unzip", "-q", src, "-d", targetDir)
	case "7z":
		cmdErr = run("7z", "x", "-bb0", "-bd", src, "-o"+targetDir)
	default:
		return Result{Err: fmt.Errorf("unknown format: %s", path)}
	}
	if cmdErr != nil {
		return Result{Err: cmdErr}
	}
	return Result{Output: "Extracted to " + targetDir}
}

// list shows archive contents.
func list(path string) Result {
	src, err := filepath.Abs(path)
	if err != nil {
		return Result{Err: err}
	}
	format := DetectFormat(src)
	var out string
	switch f := format; f {
	case "tar":
		out, err = runCapture("tar", "-tf", src)
	case "tar.gz", "tar.bz2", "tar.xz", "tar.zst":
		flag := ""
		if f == "tar.gz" {
			flag = "-z"
		} else if f == "tar.bz2" {
			flag = "-j"
		} else if f == "tar.xz" {
			flag = "-J"
		} else if f == "tar.zst" {
			flag = "--zstd"
		}
		out, err = runCapture("tar", "-tf", flag, src)
	case "zip":
		out, err = runCapture("unzip", "-l", src)
	case "7z":
		out, err = runCapture("7z", "l", src)
	default:
		return Result{Err: fmt.Errorf("unknown format: %s", path)}
	}
	if err != nil {
		return Result{Err: err}
	}
	return Result{Output: out}
}

// convert re-packs src into dest format via a temp dir.
func convert(src, dest string) Result {
	tmp, err := os.MkdirTemp("", "arx-convert-")
	if err != nil {
		return Result{Err: err}
	}
	defer os.RemoveAll(tmp)
	r := extract(src, tmp)
	if r.Err != nil {
		return r
	}
	dfmt := DetectFormat(dest)
	if dfmt == "unknown" {
		return Result{Err: fmt.Errorf("target needs a known extension: %s", dest)}
	}
	name := strings.TrimSuffix(filepath.Base(dest), filepath.Ext(dest))
	// ponytail: compress writes name+dest-ext; rename to exact dest if needed
	td := filepath.Dir(dest)
	res := compress(dfmt, name, tmp, td, 3)
	if res.Err != nil {
		return res
	}
	got := filepath.Join(td, name+"."+dfmt)
	if got != dest {
		os.Rename(got, dest)
	}
	return Result{Output: "Converted to " + dest}
}
