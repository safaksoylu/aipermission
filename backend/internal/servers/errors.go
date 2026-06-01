package servers

import "errors"

var ErrNotFound = errors.New("server not found")

type ValidationError string

func (e ValidationError) Error() string {
	return string(e)
}
