package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type outputRoundTrip func(*http.Request) (*http.Response, error)

func (f outputRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

var testPNG = append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 600)...)

type outputMemoryStore struct{ tasks map[string]*ImageTaskRecord }

func (s *outputMemoryStore) Save(_ context.Context, t *ImageTaskRecord, _ time.Duration) error {
	cp := *t
	s.tasks[t.ID] = &cp
	return nil
}
func (s *outputMemoryStore) Get(_ context.Context, id string) (*ImageTaskRecord, error) {
	t := s.tasks[id]
	if t == nil {
		return nil, ErrImageTaskNotFound
	}
	cp := *t
	return &cp, nil
}

func newOutputService(t *testing.T, result string, enabled bool, rt http.RoundTripper) *ImageTaskService {
	t.Helper()
	status := ImageTaskStatusCompleted
	if result == "PROCESSING" {
		status = ImageTaskStatusProcessing
		result = ""
	}
	store := &outputMemoryStore{tasks: map[string]*ImageTaskRecord{
		"task_1": {ID: "task_1", UserID: 7, APIKeyID: 9, Status: status, Result: jsonRaw(result)},
	}}
	uploader := NewImageResultUploader(nil, "", 1<<20, &http.Client{Transport: rt})
	return NewImageTaskServiceWithResolver(store, func() (*ImageResultUploader, bool) { return uploader, enabled }, time.Hour, time.Minute)
}

func jsonRaw(s string) []byte {
	if s == "" {
		return nil
	}
	return []byte(s)
}

func okPNG(rt func(*http.Request) (*http.Response, error)) outputRoundTrip {
	if rt != nil {
		return outputRoundTrip(rt)
	}
	return outputRoundTrip(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"image/png"}}, Body: io.NopCloser(strings.NewReader(string(testPNG)))}, nil
	})
}

var outputOwner = ImageTaskOwner{UserID: 7, APIKeyID: 9}

func TestImageTaskOutputDownload(t *testing.T) {
	var calls atomic.Int64
	var gotAuth, gotCookie, gotQuery, gotScheme atomic.Value
	// Plain-HTTP storage URL: the whole point of the proxy is fetching origins
	// a browser on HTTPS cannot reach.
	rt := outputRoundTrip(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		gotAuth.Store(r.Header.Get("Authorization"))
		gotCookie.Store(r.Header.Get("Cookie"))
		gotQuery.Store(r.URL.RawQuery)
		gotScheme.Store(r.URL.Scheme)
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"image/png"}}, Body: io.NopCloser(strings.NewReader(string(testPNG)))}, nil
	})
	svc := newOutputService(t, `{"data":[{"url":"http://cdn.example.test/a.png?sig=abc"}]}`, true, rt)

	data, ct, err := svc.GetOutput(context.Background(), outputOwner, "task_1", 0)
	require.NoError(t, err)
	require.Equal(t, "image/png", ct)
	require.Equal(t, testPNG, data)
	require.Equal(t, "http", gotScheme.Load().(string))
	require.Equal(t, "sig=abc", gotQuery.Load().(string), "signed query must reach origin unchanged")
	require.Empty(t, gotAuth.Load().(string), "no bearer leaks outbound")
	require.Empty(t, gotCookie.Load().(string))

	_, _, err = svc.GetOutput(context.Background(), ImageTaskOwner{UserID: 8, APIKeyID: 9}, "task_1", 0)
	require.ErrorIs(t, err, ErrImageTaskNotFound)
	_, _, err = svc.GetOutput(context.Background(), ImageTaskOwner{UserID: 7, APIKeyID: 10}, "task_1", 0)
	require.ErrorIs(t, err, ErrImageTaskNotFound)
	require.EqualValues(t, 1, calls.Load(), "foreign owner must not trigger any outbound request")
}

func TestImageTaskOutputStatuses(t *testing.T) {
	rt := okPNG(nil)
	owner := outputOwner

	_, _, err := newOutputService(t, "PROCESSING", true, rt).GetOutput(context.Background(), owner, "task_1", 0)
	require.ErrorIs(t, err, ErrImageTaskOutputNotReady)

	svc := newOutputService(t, `{"data":[{"url":"http://a/0.png"},{"url":"http://a/1.png"}]}`, true, rt)
	_, _, err = svc.GetOutput(context.Background(), owner, "task_1", -1)
	require.ErrorIs(t, err, ErrImageTaskOutputIndexInvalid)
	_, _, err = svc.GetOutput(context.Background(), owner, "task_1", 5)
	require.ErrorIs(t, err, ErrImageTaskOutputNotFound)

	// Top-level url serves index 0 only; empty entries never reshuffle.
	top := newOutputService(t, `{"url":"http://a/t.png"}`, true, rt)
	_, _, err = top.GetOutput(context.Background(), owner, "task_1", 0)
	require.NoError(t, err)
	_, _, err = top.GetOutput(context.Background(), owner, "task_1", 1)
	require.ErrorIs(t, err, ErrImageTaskOutputNotFound)

	// Present-but-empty data must NOT fall back to the top-level url.
	emptyData := newOutputService(t, `{"data":[],"url":"http://a/t.png"}`, true, rt)
	_, _, err = emptyData.GetOutput(context.Background(), owner, "task_1", 0)
	require.ErrorIs(t, err, ErrImageTaskOutputNotFound)

	empty := newOutputService(t, `{"data":[{"url":""},{"url":"http://a/1.png"}]}`, true, rt)
	_, _, err = empty.GetOutput(context.Background(), owner, "task_1", 0)
	require.ErrorIs(t, err, ErrImageTaskOutputNotFound)
	for _, u := range []string{`{"data":[{"url":"file:///etc/passwd"}]}`, `{"data":[{"url":"https://u:p@h/x.png"}]}`, `{"data":[{"url":"ftp://h/x.png"}]}`} {
		_, _, err = newOutputService(t, u, true, rt).GetOutput(context.Background(), owner, "task_1", 0)
		require.ErrorIs(t, err, ErrImageTaskOutputNotFound, u)
	}
}

func TestImageTaskOutputMalformedNeighbours(t *testing.T) {
	var hits atomic.Int64
	rt := outputRoundTrip(func(r *http.Request) (*http.Response, error) {
		hits.Add(1)
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"image/png"}}, Body: io.NopCloser(strings.NewReader(string(testPNG)))}, nil
	})
	// A null sibling and a primitive sibling must not break the valid entry.
	svc := newOutputService(t, `{"data":[null,123,{"url":"http://a/ok.png"}]}`, true, rt)
	data, _, err := svc.GetOutput(context.Background(), outputOwner, "task_1", 2)
	require.NoError(t, err)
	require.Equal(t, testPNG, data)
	_, _, err = svc.GetOutput(context.Background(), outputOwner, "task_1", 1)
	require.ErrorIs(t, err, ErrImageTaskOutputNotFound)
	_, _, err = svc.GetOutput(context.Background(), outputOwner, "task_1", 0)
	require.ErrorIs(t, err, ErrImageTaskOutputNotFound)
	require.EqualValues(t, 1, hits.Load())
}

func TestImageTaskOutputCallerCancel(t *testing.T) {
	rt := outputRoundTrip(func(r *http.Request) (*http.Response, error) {
		return nil, r.Context().Err()
	})
	svc := newOutputService(t, `{"data":[{"url":"http://a/x.png"}]}`, true, rt)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	data, _, err := svc.GetOutput(ctx, outputOwner, "task_1", 0)
	require.Error(t, err)
	require.Nil(t, data, "cancelled download returns no bytes")
	require.ErrorIs(t, err, ErrImageTaskOutputUnavailable)
}

func TestImageTaskOutputRejectsForgedAndBlocked(t *testing.T) {
	owner := outputOwner
	// Declared image/png but actually HTML -> 502 (sniff wins over declaration).
	rt := outputRoundTrip(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"image/png"}}, Body: io.NopCloser(strings.NewReader("<html><body>x</body></html>"))}, nil
	})
	_, _, err := newOutputService(t, `{"data":[{"url":"http://a/fake.png"}]}`, true, rt).GetOutput(context.Background(), owner, "task_1", 0)
	require.ErrorIs(t, err, ErrImageTaskOutputUnavailable)

	// SVG bytes never pass even with a spoofed MIME.
	svg := outputRoundTrip(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"image/svg+xml"}}, Body: io.NopCloser(strings.NewReader(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg"></svg>`))}, nil
	})
	_, _, err = newOutputService(t, `{"data":[{"url":"http://a/x.svg"}]}`, true, svg).GetOutput(context.Background(), owner, "task_1", 0)
	require.ErrorIs(t, err, ErrImageTaskOutputUnavailable)

	// Over the configured cap -> 502 (test uploader capped at 1MiB).
	big := outputRoundTrip(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"image/png"}}, Body: io.NopCloser(io.LimitReader(zeroReader{}, 2<<20))}, nil
	})
	_, _, err = newOutputService(t, `{"data":[{"url":"http://a/big.png"}]}`, true, big).GetOutput(context.Background(), owner, "task_1", 0)
	require.ErrorIs(t, err, ErrImageTaskOutputUnavailable)

	// SSRF guard intact: default client refuses metadata endpoints.
	svc := NewImageTaskServiceWithResolver(&outputMemoryStore{tasks: map[string]*ImageTaskRecord{
		"task_1": {ID: "task_1", UserID: 7, APIKeyID: 9, Status: ImageTaskStatusCompleted, Result: jsonRaw(`{"data":[{"url":"http://169.254.169.254/latest/meta-data"}]}`)},
	}}, func() (*ImageResultUploader, bool) { return NewImageResultUploader(nil, "", 0, nil), true }, time.Hour, time.Minute)
	_, _, err = svc.GetOutput(context.Background(), owner, "task_1", 0)
	require.ErrorIs(t, err, ErrImageTaskOutputUnavailable)
	require.NotErrorIs(t, err, ErrImageTaskNotFound)

	// Feature disabled after completion: existing tasks still download.
	svc2 := newOutputService(t, `{"data":[{"url":"http://a/ok.png"}]}`, false, okPNG(nil))
	_, _, err = svc2.GetOutput(context.Background(), owner, "task_1", 0)
	require.NoError(t, err, "disabled flag must not strand finished outputs")

	// Upstream failure maps to generic 502 without leaking the signed URL.
	fail := outputRoundTrip(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 403, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("denied"))}, nil
	})
	_, _, err = newOutputService(t, `{"data":[{"url":"http://a/x.png?sig=SECRET"}]}`, true, fail).GetOutput(context.Background(), owner, "task_1", 0)
	require.ErrorIs(t, err, ErrImageTaskOutputUnavailable)
	require.False(t, strings.Contains(err.Error(), "SECRET"))
	require.False(t, errors.Is(err, ErrImageTaskOutputNotFound))
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) { return len(p), nil }
