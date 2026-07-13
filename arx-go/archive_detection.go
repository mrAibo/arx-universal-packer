package main

import (
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"io"
	"os"
	"os/exec"
	"strconv"
)

var emptyTarBlock [512]byte

// DetectArchiveFormat identifies an existing archive by content first and
// falls back to its extension for corrupt or temporarily inaccessible files.
func DetectArchiveFormat(path string) string {
	if format := detectArchiveMagic(path); format != "unknown" {
		return format
	}
	return DetectFormat(path)
}

func detectArchiveMagic(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return "unknown"
	}
	defer file.Close()

	header := make([]byte, 512)
	read, _ := io.ReadFull(file, header)
	header = header[:read]

	switch {
	case hasSignature(header, []byte{'P', 'K', 0x03, 0x04}),
		hasSignature(header, []byte{'P', 'K', 0x05, 0x06}),
		hasSignature(header, []byte{'P', 'K', 0x07, 0x08}):
		return "zip"
	case hasSignature(header, []byte{0x37, 0x7a, 0xbc, 0xaf, 0x27, 0x1c}):
		return "7z"
	case looksLikeTarHeader(header):
		return "tar"
	case hasSignature(header, []byte{0x1f, 0x8b}):
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return "unknown"
		}
		reader, err := gzip.NewReader(file)
		if err != nil {
			return "unknown"
		}
		isTar := readerHasTarHeader(reader)
		_ = reader.Close()
		if isTar {
			return "tar.gz"
		}
	case hasSignature(header, []byte{'B', 'Z', 'h'}):
		if _, err := file.Seek(0, io.SeekStart); err == nil && readerHasTarHeader(bzip2.NewReader(file)) {
			return "tar.bz2"
		}
	case hasSignature(header, []byte{0xfd, 0x37, 0x7a, 0x58, 0x5a, 0x00}):
		if commandHasTarHeader("xz", "-dc", "--", path) {
			return "tar.xz"
		}
	case hasSignature(header, []byte{0x28, 0xb5, 0x2f, 0xfd}):
		if commandHasTarHeader("zstd", "-qdc", "--", path) {
			return "tar.zst"
		}
	}
	return "unknown"
}

func hasSignature(data, signature []byte) bool {
	return len(data) >= len(signature) && bytes.Equal(data[:len(signature)], signature)
}

func readerHasTarHeader(reader io.Reader) bool {
	header := make([]byte, 512)
	_, err := io.ReadFull(reader, header)
	return err == nil && looksLikeTarHeader(header)
}

func commandHasTarHeader(name string, args ...string) bool {
	if _, err := exec.LookPath(name); err != nil {
		return false
	}
	command := exec.Command(name, args...)
	stdout, err := command.StdoutPipe()
	if err != nil {
		return false
	}
	command.Stderr = io.Discard
	if err := command.Start(); err != nil {
		return false
	}

	header := make([]byte, 512)
	_, readErr := io.ReadFull(stdout, header)
	_ = stdout.Close()
	if command.Process != nil {
		_ = command.Process.Kill()
	}
	_ = command.Wait()
	return readErr == nil && looksLikeTarHeader(header)
}

func looksLikeTarHeader(header []byte) bool {
	if len(header) < 512 || bytes.Equal(header[:512], emptyTarBlock[:]) {
		return false
	}
	stored, ok := tarChecksum(header[148:156])
	if !ok {
		return false
	}
	calculated := 0
	for index, value := range header[:512] {
		if index >= 148 && index < 156 {
			calculated += int(' ')
		} else {
			calculated += int(value)
		}
	}
	return stored == calculated
}

func tarChecksum(field []byte) (int, bool) {
	field = bytes.Trim(field, " \x00")
	if len(field) == 0 {
		return 0, false
	}
	value, err := strconv.ParseInt(string(field), 8, 64)
	return int(value), err == nil
}
