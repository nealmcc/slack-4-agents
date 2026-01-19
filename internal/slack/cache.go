package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// FileResponseWriter writes response data to files on disk
type FileResponseWriter struct {
	dir string
}

// NewFileResponseWriter creates a response writer that stores files in the given directory
func NewFileResponseWriter(dir string) *FileResponseWriter {
	return &FileResponseWriter{dir: dir}
}

// WriteJSON marshals data to JSON and writes it to a timestamped file
func (w *FileResponseWriter) WriteJSON(name string, data any) (FileRef, error) {
	filename := fmt.Sprintf("%s-%d.json", name, time.Now().UnixNano())
	filePath := filepath.Join(w.dir, filename)

	fileData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return FileRef{}, fmt.Errorf("failed to marshal data: %w", err)
	}

	if err := os.WriteFile(filePath, fileData, 0o644); err != nil {
		return FileRef{}, fmt.Errorf("failed to write file: %w", err)
	}

	lines := bytes.Count(fileData, []byte{'\n'}) + 1

	return FileRef{
		Path:  filePath,
		Name:  filename,
		Bytes: int64(len(fileData)),
		Lines: lines,
	}, nil
}
