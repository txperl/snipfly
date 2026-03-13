package platform

import "errors"

// ErrPTYUnsupported is returned when PTY is not available on the current platform.
var ErrPTYUnsupported = errors.New("PTY is not supported on this platform")
