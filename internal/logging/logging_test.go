package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupWritesTheLog(t *testing.T) {
	dir := t.TempDir()
	logger, closer := Setup(dir, true)
	logger.Info("hello", "answer", 42)
	if err := closer.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "hello") || !strings.Contains(string(data), "answer=42") {
		t.Errorf("log:\n%s", data)
	}
}

func TestRotationKeepsThreeFiles(t *testing.T) {
	dir := t.TempDir()
	w := &rotator{dir: dir, name: FileName}
	chunk := []byte(strings.Repeat("x", 64*1024) + "\n")
	for range 40 { // a little more than 2.5 MB
		if _, err := w.Write(chunk); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{FileName, FileName + ".1", FileName + ".2"} {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if info.Size() > maxSize+int64(len(chunk)) {
			t.Errorf("%s is %d bytes", name, info.Size())
		}
	}
	if _, err := os.Stat(filepath.Join(dir, FileName+".3")); err == nil {
		t.Error("a fourth file was kept")
	}
}
