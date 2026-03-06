package slack

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestReadCanvas_ByFileID(t *testing.T) {
	mock := newMockSlackServer()
	defer mock.close()

	canvasHTML := "<h1>My Canvas</h1><p>Hello <b>world</b></p>"

	// Mock files.info
	mock.addHandler("/files.info", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"ok": true,
			"file": map[string]interface{}{
				"id":                   "F123CANVAS",
				"name":                 "My Canvas",
				"title":                "My Canvas",
				"filetype":             "quip",
				"url_private_download": mock.server.URL + "/files/F123CANVAS/download",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// Mock file download
	mock.addHandler("/files/F123CANVAS/download", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(canvasHTML))
	})

	client, _, responsesDir := newTestClient(t, mock)
	defer os.RemoveAll(responsesDir)

	ctx := context.Background()
	input := ReadCanvasInput{
		FileID: "F123CANVAS",
	}

	output, err := client.ReadCanvas(ctx, input)
	if err != nil {
		t.Fatalf("ReadCanvas failed: %v", err)
	}

	if output.FileID != "F123CANVAS" {
		t.Errorf("FileID: got %q, want %q", output.FileID, "F123CANVAS")
	}

	if output.Title != "My Canvas" {
		t.Errorf("Title: got %q, want %q", output.Title, "My Canvas")
	}

	if output.File.Path == "" {
		t.Error("File.Path: got empty, want non-empty")
	}

	// Verify content was written
	data, err := os.ReadFile(output.File.Path)
	if err != nil {
		t.Fatalf("Failed to read response file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "My Canvas") {
		t.Errorf("Content should contain 'My Canvas', got %q", content)
	}
	if !strings.Contains(content, "world") {
		t.Errorf("Content should contain 'world', got %q", content)
	}
}

func TestReadCanvas_ByChannel(t *testing.T) {
	mock := newMockSlackServer()
	defer mock.close()

	// Mock conversations.info returning canvas properties
	mock.addHandler("/conversations.info", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"ok": true,
			"channel": map[string]interface{}{
				"id":   "C123456789",
				"name": "design-docs",
				"properties": map[string]interface{}{
					"canvas": map[string]interface{}{
						"file_id":  "F456CANVAS",
						"is_empty": false,
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// Mock files.info
	mock.addHandler("/files.info", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"ok": true,
			"file": map[string]interface{}{
				"id":                   "F456CANVAS",
				"name":                 "Channel Canvas",
				"title":                "Channel Canvas",
				"filetype":             "quip",
				"url_private_download": mock.server.URL + "/files/F456CANVAS/download",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// Mock file download
	mock.addHandler("/files/F456CANVAS/download", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<p>Channel canvas content</p>"))
	})

	client, _, responsesDir := newTestClient(t, mock)
	defer os.RemoveAll(responsesDir)

	ctx := context.Background()
	input := ReadCanvasInput{
		Channel: "C123456789",
	}

	output, err := client.ReadCanvas(ctx, input)
	if err != nil {
		t.Fatalf("ReadCanvas failed: %v", err)
	}

	if output.FileID != "F456CANVAS" {
		t.Errorf("FileID: got %q, want %q", output.FileID, "F456CANVAS")
	}

	if output.Title != "Channel Canvas" {
		t.Errorf("Title: got %q, want %q", output.Title, "Channel Canvas")
	}

	data, err := os.ReadFile(output.File.Path)
	if err != nil {
		t.Fatalf("Failed to read response file: %v", err)
	}

	if !strings.Contains(string(data), "Channel canvas content") {
		t.Errorf("Content should contain 'Channel canvas content', got %q", string(data))
	}
}

func TestReadCanvas_ChannelWithoutCanvas(t *testing.T) {
	mock := newMockSlackServer()
	defer mock.close()

	// Mock conversations.info with no canvas
	mock.addHandler("/conversations.info", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"ok": true,
			"channel": map[string]interface{}{
				"id":   "C123456789",
				"name": "general",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	client, _, responsesDir := newTestClient(t, mock)
	defer os.RemoveAll(responsesDir)

	ctx := context.Background()
	input := ReadCanvasInput{
		Channel: "C123456789",
	}

	_, err := client.ReadCanvas(ctx, input)
	if err == nil {
		t.Fatal("Expected error for channel without canvas, got nil")
	}

	if !strings.Contains(err.Error(), "no canvas") {
		t.Errorf("Error should mention 'no canvas', got %q", err.Error())
	}
}

func TestReadCanvas_ValidationError(t *testing.T) {
	mock := newMockSlackServer()
	defer mock.close()

	client, _, responsesDir := newTestClient(t, mock)
	defer os.RemoveAll(responsesDir)

	ctx := context.Background()

	// Neither channel nor file_id provided
	_, err := client.ReadCanvas(ctx, ReadCanvasInput{})
	if err == nil {
		t.Fatal("Expected error when neither channel nor file_id provided, got nil")
	}

	// Both channel and file_id provided
	_, err = client.ReadCanvas(ctx, ReadCanvasInput{
		Channel: "C123456789",
		FileID:  "F123CANVAS",
	})
	if err == nil {
		t.Fatal("Expected error when both channel and file_id provided, got nil")
	}
}

func TestReadCanvas_NonCanvasFile(t *testing.T) {
	mock := newMockSlackServer()
	defer mock.close()

	// Mock files.info returning a non-canvas file
	mock.addHandler("/files.info", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"ok": true,
			"file": map[string]interface{}{
				"id":                   "F789PDF",
				"name":                 "document.pdf",
				"title":                "Some PDF",
				"filetype":             "pdf",
				"url_private_download": mock.server.URL + "/files/F789PDF/download",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	client, _, responsesDir := newTestClient(t, mock)
	defer os.RemoveAll(responsesDir)

	ctx := context.Background()
	input := ReadCanvasInput{
		FileID: "F789PDF",
	}

	_, err := client.ReadCanvas(ctx, input)
	if err == nil {
		t.Fatal("Expected error for non-canvas file, got nil")
	}

	if !strings.Contains(err.Error(), "not a canvas") {
		t.Errorf("Error should mention 'not a canvas', got %q", err.Error())
	}
}
