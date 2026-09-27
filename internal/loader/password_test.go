package loader

import (
	"errors"
	"testing"

	"github.com/bodgit/sevenzip"
	"github.com/mholt/archives"
	"github.com/nwaples/rardecode/v2"
)

func TestWithPassword(t *testing.T) {
	z := withPassword(archives.SevenZip{}, "secret").(archives.SevenZip)
	if z.Password != "secret" {
		t.Fatalf("SevenZip password = %q, want secret", z.Password)
	}

	r := withPassword(archives.Rar{}, "secret").(archives.Rar)
	if r.Password != "secret" {
		t.Fatalf("Rar password = %q, want secret", r.Password)
	}

	if _, ok := withPassword(archives.Zip{}, "secret").(archives.Zip); !ok {
		t.Fatal("Zip format should be unchanged")
	}
	if _, ok := withPassword(archives.Tar{}, "secret").(archives.Tar); !ok {
		t.Fatal("Tar format should be unchanged")
	}
}

func TestClassifyPasswordError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		hadPassword bool
		want        error
	}{
		{"rar archive encrypted", rardecode.ErrArchiveEncrypted, false, ErrPasswordRequired},
		{"rar file encrypted", rardecode.ErrArchivedFileEncrypted, false, ErrPasswordRequired},
		{"rar bad password", rardecode.ErrBadPassword, true, ErrBadPassword},
		{
			"7z encrypted no password",
			&sevenzip.ReadError{Encrypted: true, Err: errors.New("decode")},
			false,
			ErrPasswordRequired,
		},
		{
			"7z encrypted wrong password",
			&sevenzip.ReadError{Encrypted: true, Err: errors.New("decode")},
			true,
			ErrBadPassword,
		},
		{"aes no password set", errors.New("aes7z: no password set"), false, ErrPasswordRequired},
		{"plain error", errors.New("no images"), false, nil},
		{"already classified", ErrPasswordRequired, false, ErrPasswordRequired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyPasswordError(tt.err, tt.hadPassword)
			if tt.want == nil {
				if IsPasswordError(got) {
					t.Fatalf("got password error %v, want original", got)
				}
				if !errors.Is(got, tt.err) {
					t.Fatalf("got %v, want %v", got, tt.err)
				}
				return
			}
			if !errors.Is(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsPasswordError(t *testing.T) {
	if !IsPasswordError(ErrPasswordRequired) || !IsPasswordError(ErrBadPassword) {
		t.Fatal("sentinels should be password errors")
	}
	if IsPasswordError(errors.New("other")) {
		t.Fatal("unrelated error should not be a password error")
	}
}
