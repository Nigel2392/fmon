package configure

import (
	"errors"
)

var (
	ErrConfigNotExists = errors.New("Configuration file not found.")
	ErrConfigRead      = errors.New("Error while reading config.")
	ErrConfigWrite     = errors.New("Error while writing config.")
)
