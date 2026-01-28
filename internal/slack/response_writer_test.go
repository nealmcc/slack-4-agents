package slack

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteJSONLines_Basic(t *testing.T) {
	dir, err := os.MkdirTemp("", "response-writer-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	w := NewFileResponseWriter(dir)

	type testData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	ref, err := w.WriteJSONLines("test", func(jw JSONLineWriter) error {
		if err := jw.WriteLine(testData{Name: "first", Value: 1}); err != nil {
			return err
		}
		if err := jw.WriteLine(testData{Name: "second", Value: 2}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WriteJSONLines failed: %v", err)
	}

	if ref.Lines != 2 {
		t.Errorf("Lines: got %d, want 2", ref.Lines)
	}

	if !strings.HasSuffix(ref.Name, ".jsonl") {
		t.Errorf("Name: got %q, want .jsonl suffix", ref.Name)
	}

	data, err := os.ReadFile(ref.Path)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("File lines: got %d, want 2", len(lines))
	}

	var first testData
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("Failed to unmarshal first line: %v", err)
	}
	if first.Name != "first" || first.Value != 1 {
		t.Errorf("First line: got %+v, want {Name:first Value:1}", first)
	}

	var second testData
	if err := json.Unmarshal([]byte(lines[1]), &second); err != nil {
		t.Fatalf("Failed to unmarshal second line: %v", err)
	}
	if second.Name != "second" || second.Value != 2 {
		t.Errorf("Second line: got %+v, want {Name:second Value:2}", second)
	}
}

func TestWriteJSONLines_Empty(t *testing.T) {
	dir, err := os.MkdirTemp("", "response-writer-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	w := NewFileResponseWriter(dir)

	ref, err := w.WriteJSONLines("empty", func(jw JSONLineWriter) error {
		return nil
	})
	if err != nil {
		t.Fatalf("WriteJSONLines failed: %v", err)
	}

	if ref.Lines != 0 {
		t.Errorf("Lines: got %d, want 0", ref.Lines)
	}

	if ref.Bytes != 0 {
		t.Errorf("Bytes: got %d, want 0", ref.Bytes)
	}

	data, err := os.ReadFile(ref.Path)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if len(data) != 0 {
		t.Errorf("File content: got %q, want empty", string(data))
	}
}

func TestWriteJSONLines_WriterError(t *testing.T) {
	dir, err := os.MkdirTemp("", "response-writer-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	w := NewFileResponseWriter(dir)

	wantErr := errors.New("write callback error")
	_, err = w.WriteJSONLines("error", func(jw JSONLineWriter) error {
		return wantErr
	})
	if err != wantErr {
		t.Errorf("Error: got %v, want %v", err, wantErr)
	}
}

func TestWriteJSONLines_MarshalError(t *testing.T) {
	dir, err := os.MkdirTemp("", "response-writer-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	w := NewFileResponseWriter(dir)

	_, err = w.WriteJSONLines("marshal-error", func(jw JSONLineWriter) error {
		return jw.WriteLine(make(chan int))
	})
	if err == nil {
		t.Error("Expected marshal error, got nil")
	}
}

func TestWriteJSONLines_DirectoryNotExist(t *testing.T) {
	w := NewFileResponseWriter("/nonexistent/path/that/does/not/exist")

	_, err := w.WriteJSONLines("test", func(jw JSONLineWriter) error {
		return jw.WriteLine("data")
	})
	if err == nil {
		t.Error("Expected error for nonexistent directory, got nil")
	}
}

func TestWriteJSON(t *testing.T) {
	dir, err := os.MkdirTemp("", "response-writer-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	w := NewFileResponseWriter(dir)

	type testData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	ref, err := w.WriteJSON("test", testData{Name: "test", Value: 42})
	if err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	if !strings.HasSuffix(ref.Name, ".json") {
		t.Errorf("Name: got %q, want .json suffix", ref.Name)
	}

	if !filepath.IsAbs(ref.Path) {
		t.Errorf("Path is not absolute: %s", ref.Path)
	}

	data, err := os.ReadFile(ref.Path)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	var result testData
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if result.Name != "test" || result.Value != 42 {
		t.Errorf("Data: got %+v, want {Name:test Value:42}", result)
	}
}
