package tokens

import "errors"

var ErrNotFound = errors.New("token not found")

type ValidationError string

func (e ValidationError) Error() string {
	return string(e)
}
