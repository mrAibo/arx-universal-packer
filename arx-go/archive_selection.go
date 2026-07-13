package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func compressMany(format, name string, sources []string, baseDir, targetDir string, level int) Result {
	if len(sources) == 0 {
		return Result{Err: fmt.Errorf("no files or directories selected")}
	}
	if level < 1 {
		level = 1
	}
	if level > 9 {
		level = 9
	}

	baseAbs, err := filepath.Abs(baseDir)
	if err != nil {
		return Result{Err: err}
	}
	targetAbs, err := filepath.Abs(targetDir)
	if err != nil {
		return Result{Err: err}
	}
	if err := os.MkdirAll(targetAbs, 0o755); err != nil {
		return Result{Err: err}
	}

	relative, err := relativeSources(baseAbs, sources)
	if err != nil {
		return Result{Err: err}
	}
	out := filepath.Join(targetAbs, name+"."+format)
	if _, err := os.Stat(out); err == nil {
		return Result{Err: fmt.Errorf("archive already exists: %s", out)}
	} else if !os.IsNotExist(err) {
		return Result{Err: err}
	}

	var cmdErr error
	switch format {
	case "tar":
		args := append([]string{"-cf", out, "--"}, relative...)
		cmdErr = runIn(baseAbs, "tar", args...)
	case "tar.gz":
		compressor := "gzip"
		if _, lookupErr := exec.LookPath("pigz"); lookupErr == nil {
			compressor = "pigz"
		}
		cmdErr = pipeTarMany(out, baseAbs, relative, compressor, fmt.Sprintf("-%d", level))
	case "tar.bz2":
		compressor := "bzip2"
		if _, lookupErr := exec.LookPath("pbzip2"); lookupErr == nil {
			compressor = "pbzip2"
		}
		cmdErr = pipeTarMany(out, baseAbs, relative, compressor, fmt.Sprintf("-%d", level))
	case "tar.xz":
		args := append([]string{"-cJf", out, "--"}, relative...)
		cmd := exec.Command("tar", args...)
		cmd.Dir = baseAbs
		cmd.Env = append(os.Environ(), fmt.Sprintf("XZ_OPT=--threads=0 -%d", level))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmdErr = cmd.Run()
	case "tar.zst":
		if _, lookupErr := exec.LookPath("zstd"); lookupErr == nil {
			cmdErr = pipeTarMany(out, baseAbs, relative, "zstd", fmt.Sprintf("-%d", level))
		} else {
			args := append([]string{"--zstd", "-cf", out, "--"}, relative...)
			cmdErr = runIn(baseAbs, "tar", args...)
		}
	case "zip":
		args := []string{"-r", fmt.Sprintf("-%d", level), out, "-@"}
		cmd := exec.Command("zip", args...)
		cmd.Dir = baseAbs
		cmd.Stdin = strings.NewReader(strings.Join(relative, "\n") + "\n")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmdErr = cmd.Run()
	case "7z":
		listFile, listErr := writeListFile(relative)
		if listErr != nil {
			return Result{Err: listErr}
		}
		defer os.Remove(listFile)
		cmd := exec.Command("7z", "a", fmt.Sprintf("-mx=%d", level), "-scsUTF-8", out, "@"+listFile)
		cmd.Dir = baseAbs
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmdErr = cmd.Run()
	default:
		return Result{Err: fmt.Errorf("unsupported format: %s", format)}
	}
	if cmdErr != nil {
		_ = os.Remove(out)
		return Result{Err: cmdErr}
	}
	return Result{Output: fmt.Sprintf("Archive created: %s (%d selected item(s))", out, len(relative))}
}

func relativeSources(baseDir string, sources []string) ([]string, error) {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(sources))
	for _, source := range sources {
		absolute, err := filepath.Abs(source)
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(absolute); err != nil {
			return nil, fmt.Errorf("source not found: %s", source)
		}
		relative, err := filepath.Rel(baseDir, absolute)
		if err != nil {
			return nil, err
		}
		if relative == "." || relative == ".." || filepath.IsAbs(relative) || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
			return nil, fmt.Errorf("source is outside the active panel: %s", source)
		}
		relative = filepath.ToSlash(relative)
		if _, ok := seen[relative]; ok {
			continue
		}
		seen[relative] = struct{}{}
		result = append(result, relative)
	}
	sort.Strings(result)
	return result, nil
}

func pipeTarMany(out, baseDir string, sources []string, compressor, level string) error {
	args := append([]string{"-c", "--"}, sources...)
	tarCmd := exec.Command("tar", args...)
	tarCmd.Dir = baseDir
	compressCmd := exec.Command(compressor, level, "-c")

	reader, writer := io.Pipe()
	tarCmd.Stdout = writer
	tarCmd.Stderr = os.Stderr
	compressCmd.Stdin = reader
	compressCmd.Stderr = os.Stderr

	output, err := os.Create(out)
	if err != nil {
		return err
	}
	defer output.Close()
	compressCmd.Stdout = output

	if err := tarCmd.Start(); err != nil {
		return err
	}
	if err := compressCmd.Start(); err != nil {
		_ = writer.CloseWithError(err)
		_ = tarCmd.Process.Kill()
		return err
	}

	tarDone := make(chan error, 1)
	go func() {
		err := tarCmd.Wait()
		_ = writer.CloseWithError(err)
		tarDone <- err
	}()

	compressErr := compressCmd.Wait()
	tarErr := <-tarDone
	if tarErr != nil {
		return tarErr
	}
	return compressErr
}

func extractSelected(archivePath string, members []string, targetDir string) Result {
	source, err := filepath.Abs(archivePath)
	if err != nil {
		return Result{Err: err}
	}
	if _, err := os.Stat(source); err != nil {
		return Result{Err: fmt.Errorf("archive not found: %s", archivePath)}
	}
	target, err := filepath.Abs(targetDir)
	if err != nil {
		return Result{Err: err}
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		return Result{Err: err}
	}

	if len(members) == 0 {
		members, err = readArchiveItems(source)
		if err != nil {
			return Result{Err: err}
		}
	}
	members = safeUniqueMembers(members)
	if len(members) == 0 {
		return Result{Err: fmt.Errorf("archive has no safe extractable entries")}
	}

	format := DetectFormat(strings.ToLower(source))
	var cmdErr error
	switch format {
	case "tar", "tar.gz", "tar.bz2", "tar.xz", "tar.zst":
		listFile, listErr := writeListFile(members)
		if listErr != nil {
			return Result{Err: listErr}
		}
		defer os.Remove(listFile)
		args := tarExtractArgs(format, source, target, listFile)
		cmdErr = run("tar", args...)
	case "zip":
		args := append([]string{"-q", source}, members...)
		args = append(args, "-d", target)
		cmdErr = run("unzip", args...)
	case "7z":
		listFile, listErr := writeListFile(members)
		if listErr != nil {
			return Result{Err: listErr}
		}
		defer os.Remove(listFile)
		cmdErr = run("7z", "x", "-bb0", "-bd", "-scsUTF-8", "-o"+target, source, "@"+listFile)
	default:
		return Result{Err: fmt.Errorf("unsupported archive format: %s", archivePath)}
	}
	if cmdErr != nil {
		return Result{Err: cmdErr}
	}
	return Result{Output: fmt.Sprintf("Extracted %d archive item(s) to %s", len(members), target)}
}

func tarExtractArgs(format, source, target, listFile string) []string {
	args := []string{"-x"}
	switch format {
	case "tar.gz":
		args = append(args, "-z")
	case "tar.bz2":
		args = append(args, "-j")
	case "tar.xz":
		args = append(args, "-J")
	case "tar.zst":
		args = append(args, "--zstd")
	}
	return append(args, "-f", source, "-C", target, "--verbatim-files-from", "-T", listFile)
}

func safeUniqueMembers(values []string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(values))
	for _, value := range values {
		member := normalizeArchivePath(value)
		if member == "" {
			continue
		}
		if _, ok := seen[member]; ok {
			continue
		}
		seen[member] = struct{}{}
		result = append(result, member)
	}
	sort.Strings(result)
	return result
}

func writeListFile(values []string) (string, error) {
	file, err := os.CreateTemp("", "arx-list-*.txt")
	if err != nil {
		return "", err
	}
	name := file.Name()
	for _, value := range values {
		if strings.ContainsAny(value, "\r\n") {
			_ = file.Close()
			_ = os.Remove(name)
			return "", fmt.Errorf("file names containing line breaks are not supported")
		}
		if _, err := fmt.Fprintln(file, value); err != nil {
			_ = file.Close()
			_ = os.Remove(name)
			return "", err
		}
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(name)
		return "", err
	}
	return name, nil
}

func testArchive(path string) Result {
	format := DetectFormat(strings.ToLower(path))
	var output string
	var err error
	switch format {
	case "tar":
		output, err = runCapture("tar", "-tf", path)
	case "tar.gz":
		output, err = runCapture("tar", "-tzf", path)
	case "tar.bz2":
		output, err = runCapture("tar", "-tjf", path)
	case "tar.xz":
		output, err = runCapture("tar", "-tJf", path)
	case "tar.zst":
		output, err = runCapture("tar", "--zstd", "-tf", path)
	case "zip":
		output, err = runCapture("unzip", "-tq", path)
	case "7z":
		output, err = runCapture("7z", "t", "-bb0", "-bd", path)
	default:
		return Result{Err: fmt.Errorf("unsupported archive format: %s", path)}
	}
	if err != nil {
		message := strings.TrimSpace(output)
		if message == "" {
			message = err.Error()
		}
		return Result{Err: fmt.Errorf("archive test failed: %s", message)}
	}
	return Result{Output: "Archive test passed: " + path}
}

func convertArchive(source, destination string, level int) Result {
	format := DetectFormat(strings.ToLower(destination))
	if format == "unknown" {
		return Result{Err: fmt.Errorf("target needs a known extension: %s", destination)}
	}
	temporary, err := os.MkdirTemp("", "arx-convert-")
	if err != nil {
		return Result{Err: err}
	}
	defer os.RemoveAll(temporary)

	extracted := extractSelected(source, nil, temporary)
	if extracted.Err != nil {
		return extracted
	}
	items, err := os.ReadDir(temporary)
	if err != nil {
		return Result{Err: err}
	}
	if len(items) == 0 {
		return Result{Err: fmt.Errorf("cannot convert an empty archive")}
	}
	sources := make([]string, 0, len(items))
	for _, item := range items {
		sources = append(sources, filepath.Join(temporary, item.Name()))
	}
	name := defaultOutputName(destination)
	result := compressMany(format, name, sources, temporary, filepath.Dir(destination), level)
	if result.Err != nil {
		return result
	}
	return Result{Output: "Converted to " + destination}
}
