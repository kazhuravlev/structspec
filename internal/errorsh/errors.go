package errorsh

import "fmt"

// Wrap helps to wrap errors.
func Wrap(err error, msg string) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s: %w", msg, err)
}

func Wrapf(err error, msgFmt string, args ...any) error {
	return Wrap(err, fmt.Sprintf(msgFmt, args...))
}

func Newf(msgFmt string, args ...any) error {
	return fmt.Errorf(msgFmt, args...)
}
