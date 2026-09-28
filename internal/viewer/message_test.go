package viewer

import (
	"testing"

	"github.com/henrylau/zip-photo-viewer/internal/loader"
)

func TestShowNoMedia(t *testing.T) {
	v := NewViewer(nil, "")
	v.ShowNoMedia("/photos/album.zip")
	if v.message == nil {
		t.Fatal("expected message prompt")
	}
	if v.message.title != "No media found" {
		t.Errorf("title = %q", v.message.title)
	}
	want := "album.zip does not contain any supported images."
	if v.message.body != want {
		t.Errorf("body = %q, want %q", v.message.body, want)
	}
	if !v.dismissMessage() {
		t.Fatal("dismissMessage() should return true")
	}
	if v.message != nil {
		t.Fatal("message should be cleared")
	}
	if v.dismissMessage() {
		t.Fatal("dismissMessage() should return false when no message")
	}
}

func TestHandleOpenError_NoMedia(t *testing.T) {
	v := NewViewer(nil, "")
	handleOpenError(&v, "/photos/empty.zip", loader.ErrNoMedia)
	if v.message == nil {
		t.Fatal("expected no-media prompt")
	}
	if v.prompt != nil {
		t.Fatal("password prompt should stay closed")
	}
}

func TestHandleOpenError_Password(t *testing.T) {
	v := NewViewer(nil, "")
	handleOpenError(&v, "/photos/secret.7z", loader.ErrPasswordRequired)
	if v.prompt == nil {
		t.Fatal("expected password prompt")
	}
	if v.message != nil {
		t.Fatal("no-media prompt should stay closed")
	}
}
