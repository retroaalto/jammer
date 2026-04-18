//go:build windows

package audio

import "errors"

type BassBackend struct{}

func LoadBass(_ string) (Backend, error) {
	return nil, errors.New("BASS backend is not supported on Windows")
}
