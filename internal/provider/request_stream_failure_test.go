package provider

import (
	"context"
	"errors"
	"testing"
)

func TestStreamOrRequestErrorPreservesDefiniteFailure(t *testing.T) {
	want := errors.New("definite failure")
	stream, err := StreamOrRequestError(context.Background(), want)
	if stream != nil || !errors.Is(err, want) {
		t.Fatalf("stream = %v, err = %v, want nil stream and original error", stream, err)
	}
}

func TestStreamOrRequestErrorDefersAmbiguousFailureToAgent(t *testing.T) {
	want := NewRequestError("relay", RequestFailureNetwork, errors.New("connection reset"))
	want.RequestMayHaveReachedServer = true
	stream, err := StreamOrRequestError(context.Background(), want)
	if err != nil || stream == nil {
		t.Fatalf("stream = %v, err = %v, want stream and nil error", stream, err)
	}
	chunk, ok := <-stream
	if !ok || chunk.Type != ChunkError || chunk.Err == nil {
		t.Fatalf("chunk = %#v, ok = %v, want one stream error", chunk, ok)
	}
}
