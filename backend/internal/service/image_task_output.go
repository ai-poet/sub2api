package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Errors for the task-output content proxy. Messages stay generic so callers
// never learn a signed upstream URL or storage key from a 502.
var (
	ErrImageTaskOutputNotReady     = infraerrors.New(http.StatusConflict, "IMAGE_TASK_OUTPUT_NOT_READY", "image task output is not ready")
	ErrImageTaskOutputIndexInvalid = infraerrors.New(http.StatusBadRequest, "IMAGE_TASK_OUTPUT_INDEX_INVALID", "image task output index is invalid")
	ErrImageTaskOutputNotFound     = infraerrors.New(http.StatusNotFound, "IMAGE_TASK_OUTPUT_NOT_FOUND", "image task output not found")
	ErrImageTaskOutputUnavailable  = infraerrors.New(http.StatusBadGateway, "IMAGE_TASK_OUTPUT_UNAVAILABLE", "image task output is unavailable")
)

// imageTaskResultURLs mirrors the upstream image response payload persisted in
// task.Result. Data stays raw: entries keep their original index and only the
// requested entry is decoded, so a malformed sibling can never take down a
// healthy output or reshuffle the list.
type imageTaskResultURLs struct {
	Data json.RawMessage `json:"data"`
	URL  string          `json:"url"`
}

// outputURLAt returns the remote URL stored at data[index]. A present-but-empty
// data array never falls back to the top-level url (matching the frontend's
// extractTaskOutputs); a missing or null data field does, but only at index 0.
func outputURLAt(result json.RawMessage, index int) (string, error) {
	var parsed imageTaskResultURLs
	if err := json.Unmarshal(result, &parsed); err != nil {
		return "", ErrImageTaskOutputUnavailable.WithCause(err)
	}
	if len(parsed.Data) > 0 && string(parsed.Data) != "null" {
		var entries []json.RawMessage
		if err := json.Unmarshal(parsed.Data, &entries); err != nil {
			return "", ErrImageTaskOutputUnavailable.WithCause(err)
		}
		if index >= len(entries) {
			return "", ErrImageTaskOutputNotFound
		}
		var entry struct {
			URL string `json:"url"`
		}
		// null, primitives and objects without a url all land on the same 404;
		// they never disturb neighbouring indexes.
		if err := json.Unmarshal(entries[index], &entry); err != nil {
			return "", ErrImageTaskOutputNotFound
		}
		return entry.URL, nil
	}
	if index == 0 {
		return parsed.URL, nil
	}
	return "", ErrImageTaskOutputNotFound
}

// GetOutput downloads one remote task output through the SSRF-guarded image
// downloader so the browser never has to reach the (possibly plain-HTTP or
// CORS-locked) origin URL. Ownership, lifetime and index validation all mirror
// Get: cross-user/key IDs are indistinguishable from missing tasks.
func (s *ImageTaskService) GetOutput(ctx context.Context, owner ImageTaskOwner, id string, index int) ([]byte, string, error) {
	if index < 0 {
		return nil, "", ErrImageTaskOutputIndexInvalid
	}
	task, err := s.Get(ctx, owner, id)
	if err != nil {
		return nil, "", err
	}
	if task.Status != ImageTaskStatusCompleted || len(task.Result) == 0 {
		return nil, "", ErrImageTaskOutputNotReady
	}
	rawURL, err := outputURLAt(task.Result, index)
	if err != nil {
		return nil, "", err
	}
	rawURL = strings.TrimSpace(rawURL)
	if !isProxyableOutputURL(rawURL) {
		return nil, "", ErrImageTaskOutputNotFound
	}
	uploader, _ := s.current()
	if uploader == nil {
		uploader = NewImageResultUploader(nil, "", defaultImageMaxDownloadBytes, nil)
	}
	// Value copy: the shared uploader keeps its pool but the proxy caps the
	// per-response read at 32MiB regardless of the configured upload limit.
	bounded := *uploader
	if bounded.maxDownloadBytes <= 0 || bounded.maxDownloadBytes > defaultImageMaxDownloadBytes {
		bounded.maxDownloadBytes = defaultImageMaxDownloadBytes
	}
	data, _, err := bounded.download(ctx, rawURL)
	if err != nil {
		return nil, "", ErrImageTaskOutputUnavailable.WithCause(err)
	}
	// The declared MIME is not trusted: bytes must sniff as a real raster image
	// so a spoofed image/* label cannot relay HTML/SVG to the tenant browser.
	switch ct := detectedImageContentType(data); ct {
	case "image/png", "image/jpeg", "image/webp", "image/gif":
		return data, ct, nil
	default:
		return nil, "", ErrImageTaskOutputUnavailable
	}
}

// isProxyableOutputURL accepts only credential-free http(s) URLs; the stored
// reference is never substituted with a caller-supplied address.
func isProxyableOutputURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u == nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	return u.Host != "" && u.User == nil
}
