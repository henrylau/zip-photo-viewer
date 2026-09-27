package loader

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bodgit/sevenzip"
	"github.com/mholt/archives"
	"github.com/nwaples/rardecode/v2"
)

var (
	ErrPasswordRequired        = errors.New("archive password required")
	ErrBadPassword             = errors.New("incorrect archive password")
	ErrEncryptedZipUnsupported = errors.New("password-protected zip is not supported")
)

func IsPasswordError(err error) bool {
	return errors.Is(err, ErrPasswordRequired) || errors.Is(err, ErrBadPassword)
}

func classifyPasswordError(err error, hadPassword bool) error {
	if err == nil {
		return nil
	}
	if mapped := mapPasswordError(err, hadPassword); mapped != nil {
		return mapped
	}
	return err
}

func mapPasswordError(err error, hadPassword bool) error {
	if errors.Is(err, ErrPasswordRequired) || errors.Is(err, ErrBadPassword) || errors.Is(err, ErrEncryptedZipUnsupported) {
		return err
	}
	if errors.Is(err, rardecode.ErrArchiveEncrypted) || errors.Is(err, rardecode.ErrArchivedFileEncrypted) {
		return fmt.Errorf("%w: %s", ErrPasswordRequired, err.Error())
	}
	if errors.Is(err, rardecode.ErrBadPassword) {
		return fmt.Errorf("%w: %s", ErrBadPassword, err.Error())
	}

	var readErr *sevenzip.ReadError
	if errors.As(err, &readErr) && readErr.Encrypted {
		if hadPassword {
			return fmt.Errorf("%w: %s", ErrBadPassword, err.Error())
		}
		return fmt.Errorf("%w: %s", ErrPasswordRequired, err.Error())
	}

	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "no password set") {
		return fmt.Errorf("%w: %s", ErrPasswordRequired, err.Error())
	}
	if strings.Contains(msg, "encrypt") || strings.Contains(msg, "password") {
		if strings.Contains(msg, "zip") {
			return fmt.Errorf("%w: %s", ErrEncryptedZipUnsupported, err.Error())
		}
		if hadPassword {
			return fmt.Errorf("%w: %s", ErrBadPassword, err.Error())
		}
		return fmt.Errorf("%w: %s", ErrPasswordRequired, err.Error())
	}
	return nil
}

func withPassword(format archives.Format, password string) archives.Format {
	switch f := format.(type) {
	case archives.SevenZip:
		f.Password = password
		return f
	case *archives.SevenZip:
		cp := *f
		cp.Password = password
		return cp
	case archives.Rar:
		f.Password = password
		return f
	case *archives.Rar:
		cp := *f
		cp.Password = password
		return cp
	default:
		return format
	}
}
