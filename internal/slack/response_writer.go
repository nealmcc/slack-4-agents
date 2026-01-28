package slack

import (
	"bufio"
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

// Dir returns the directory where files are written
func (w *FileResponseWriter) Dir() string {
	return w.dir
}

// WriteJSON marshals data to JSON and writes it to a timestamped file
func (w *FileResponseWriter) WriteJSON(name string, data any) (FileRef, error) {
	filename := fmt.Sprintf("%s-%d.json", name, time.Now().UnixNano())
	filePath := filepath.Join(w.dir, filename)

	file, err := os.Create(filePath)
	if err != nil {
		return FileRef{}, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		return FileRef{}, fmt.Errorf("failed to write data: %w", err)
	}

	fi, err := file.Stat()
	if err != nil {
		return FileRef{}, fmt.Errorf("failed to stat file: %w", err)
	}

	return FileRef{
		Path:  filePath,
		Name:  filename,
		Bytes: fi.Size(),
		Lines: 1,
	}, nil
}

// jsonLineWriter implements JSONLineWriter for streaming writes directly to disk
type jsonLineWriter struct {
	bw    *bufio.Writer
	lines int
}

func (w *jsonLineWriter) WriteLine(data any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal line: %w", err)
	}
	if _, err := w.bw.Write(b); err != nil {
		return err
	}
	if err := w.bw.WriteByte('\n'); err != nil {
		return err
	}
	w.lines++
	return nil
}

// WriteJSONLines writes data in JSON-lines format using a streaming callback.
// Data is written directly to disk via buffered I/O rather than accumulated in memory.
func (w *FileResponseWriter) WriteJSONLines(name string, writeFn func(jw JSONLineWriter) error) (FileRef, error) {
	filename := fmt.Sprintf("%s-%d.jsonl", name, time.Now().UnixNano())
	return w.writeJSONLinesFile(filename, writeFn)
}

// WriteJSONLinesNamed writes data in JSON-lines format to a file with the specified name.
// Unlike WriteJSONLines, this does not add a timestamp suffix.
func (w *FileResponseWriter) WriteJSONLinesNamed(filename string, writeFn func(jw JSONLineWriter) error) (FileRef, error) {
	return w.writeJSONLinesFile(filename, writeFn)
}

func (w *FileResponseWriter) writeJSONLinesFile(filename string, writeFn func(jw JSONLineWriter) error) (FileRef, error) {
	filePath := filepath.Join(w.dir, filename)

	file, err := os.Create(filePath)
	if err != nil {
		return FileRef{}, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	jw := &jsonLineWriter{bw: bufio.NewWriter(file)}

	if err := writeFn(jw); err != nil {
		return FileRef{}, err
	}

	if err := jw.bw.Flush(); err != nil {
		return FileRef{}, fmt.Errorf("failed to flush buffer: %w", err)
	}

	fi, err := file.Stat()
	if err != nil {
		return FileRef{}, fmt.Errorf("failed to stat file: %w", err)
	}

	return FileRef{
		Path:  filePath,
		Name:  filename,
		Bytes: fi.Size(),
		Lines: jw.lines,
	}, nil
}
