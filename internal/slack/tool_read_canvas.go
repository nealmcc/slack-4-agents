package slack

import (
	"bytes"
	"context"
	"fmt"

	"github.com/slack-go/slack"
)

// ReadCanvasInput defines input for reading a Slack canvas
type ReadCanvasInput struct {
	Channel string `json:"channel,omitempty" jsonschema:"Channel ID or name (for channel canvases)"`
	FileID  string `json:"file_id,omitempty" jsonschema:"Canvas file ID (for standalone canvases)"`
}

// ReadCanvasOutput contains the canvas content and metadata
type ReadCanvasOutput struct {
	File   FileRef `json:"file"`
	FileID string  `json:"file_id"`
	Title  string  `json:"title"`
}

// ReadCanvas reads a Slack canvas and returns its content as plain text
func (c *Service) ReadCanvas(ctx context.Context, input ReadCanvasInput) (ReadCanvasOutput, error) {
	if input.Channel == "" && input.FileID == "" {
		return ReadCanvasOutput{}, fmt.Errorf("either channel or file_id is required")
	}
	if input.Channel != "" && input.FileID != "" {
		return ReadCanvasOutput{}, fmt.Errorf("provide either channel or file_id, not both")
	}

	fileID := input.FileID

	if input.Channel != "" {
		channelID, err := c.GetChannelID(input.Channel)
		if err != nil {
			return ReadCanvasOutput{}, err
		}

		ch, err := c.getConversationInfo(ctx, channelID)
		if err != nil {
			return ReadCanvasOutput{}, fmt.Errorf("failed to get channel info: %w", err)
		}

		if ch.Properties == nil || ch.Properties.Canvas.FileId == "" {
			return ReadCanvasOutput{}, fmt.Errorf("channel has no canvas")
		}
		fileID = ch.Properties.Canvas.FileId
	}

	var file *slack.File
	err := withRetry(ctx, c.logger, func() error {
		var e error
		file, _, _, e = c.api.GetFileInfoContext(ctx, fileID, 0, 0)
		return e
	})
	if err != nil {
		return ReadCanvasOutput{}, fmt.Errorf("failed to get file info: %w", err)
	}

	if file.Filetype != "quip" {
		return ReadCanvasOutput{}, fmt.Errorf("file is not a canvas (filetype %q, expected \"quip\")", file.Filetype)
	}

	var buf bytes.Buffer
	err = withRetry(ctx, c.logger, func() error {
		buf.Reset()
		return c.api.GetFileContext(ctx, file.URLPrivateDownload, &buf)
	})
	if err != nil {
		return ReadCanvasOutput{}, fmt.Errorf("failed to download canvas: %w", err)
	}

	text := stripHTML(buf.String())

	ref, err := c.responses.WriteText("canvas", text)
	if err != nil {
		return ReadCanvasOutput{}, fmt.Errorf("failed to write response: %w", err)
	}

	return ReadCanvasOutput{
		File:   ref,
		FileID: fileID,
		Title:  file.Title,
	}, nil
}
