package mcpserver

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const maxResponseBytes = 32 << 20

func (i *instance) requestOnce(ctx context.Context, method, path string, metadata operationMetadata, args map[string]any) (*mcp.CallToolResult, error) {
	timeout := 30 * time.Second
	if metadata.Transport == "sse" || metadata.Transport == "multipart" {
		timeout = 2 * time.Hour
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var body io.Reader
	contentType := ""
	contentLength := int64(0)
	if value, ok := args["body"]; ok {
		data, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}

		body = bytes.NewReader(data)
		contentType = "application/json"
	}

	if metadata.Transport == "multipart" {
		path, _ := args["file_path"].(string)
		if !filepath.IsAbs(path) {
			return nil, fmt.Errorf("file_path must be absolute")
		}

		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}

		defer func() { _ = file.Close() }()
		info, err := file.Stat()
		if err != nil {
			return nil, err
		}

		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("upload requires a regular file")
		}

		limit := metadata.Upload.MaxBytes
		ext := strings.ToLower(filepath.Ext(path))
		if len(metadata.Upload.Extensions) > 0 && !slices.Contains(metadata.Upload.Extensions, ext) {
			return nil, fmt.Errorf("file extension must be one of %v", metadata.Upload.Extensions)
		}

		var framing bytes.Buffer
		form := multipart.NewWriter(&framing)
		if _, err := form.CreateFormFile(metadata.Upload.Field, filepath.Base(path)); err != nil {
			return nil, err
		}

		headerSize := framing.Len()
		if err := form.Close(); err != nil {
			return nil, err
		}

		contentLength = info.Size() + int64(framing.Len())
		if contentLength > limit {
			return nil, fmt.Errorf("file exceeds upload limit including multipart framing")
		}

		contentType = form.FormDataContentType()
		body = io.MultiReader(bytes.NewReader(framing.Bytes()[:headerSize]), file, bytes.NewReader(framing.Bytes()[headerSize:]))
	}

	req, err := http.NewRequestWithContext(ctx, method, i.url+path, body)
	if err != nil {
		return nil, err
	}

	if contentLength > 0 {
		req.ContentLength = contentLength
	}

	req.Header.Set("Authorization", "Bearer "+i.token)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := i.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", i.name, err)
	}

	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		return nil, fmt.Errorf("%s: HTTP %d: %s", i.name, resp.StatusCode, strings.ReplaceAll(string(data), i.token, "[redacted]"))
	}

	if metadata.Transport == "sse" {
		return readJobStream(resp.Body, *metadata.Stream)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, err
	}

	if len(data) > maxResponseBytes {
		return nil, fmt.Errorf("%s: response exceeds %d bytes", i.name, maxResponseBytes)
	}

	if len(data) == 0 {
		return jsonResult(map[string]any{"status": resp.StatusCode}), nil
	}

	mediaType, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if mediaType == "image/png" {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.ImageContent{MIMEType: mediaType, Data: data}}}, nil
	}

	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, fmt.Errorf("%s: expected JSON response: %w", i.name, err)
	}

	return jsonResult(value), nil
}

func readJobStream(reader io.Reader, metadata streamMetadata) (*mcp.CallToolResult, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4096), maxResponseBytes)
	event := ""
	var data strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if event == metadata.Event {
				var job any
				if err := json.Unmarshal([]byte(data.String()), &job); err != nil {
					return nil, fmt.Errorf("decode job: %w", err)
				}

				result := jsonResult(job)
				if obj, ok := job.(map[string]any); ok && obj[metadata.StateField] == metadata.FailureValue {
					result.IsError = true
				}

				return result, nil
			}

			event = ""
			data.Reset()
			continue
		}

		if strings.HasPrefix(line, "event:") {
			event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		}

		if strings.HasPrefix(line, "data:") {
			if data.Len() > 0 {
				data.WriteByte('\n')
			}

			_, _ = data.WriteString(strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
			if data.Len() > maxResponseBytes {
				return nil, fmt.Errorf("job event exceeds response limit")
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return nil, fmt.Errorf("job stream closed before completion; use getJob to check its state")
}
