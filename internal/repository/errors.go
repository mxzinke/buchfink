package repository

import (
	"errors"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

// mapError ergänzt die fachliche Kennung, ohne durchgereichte Fehlertexte
// oder die ursprüngliche Fehlerkette zu verändern.
func mapError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) && !errors.Is(err, domain.ErrNotFound) {
		return notFoundError{cause: err}
	}
	return err
}

type notFoundError struct{ cause error }

func (e notFoundError) Error() string        { return e.cause.Error() }
func (e notFoundError) Unwrap() error        { return e.cause }
func (e notFoundError) Is(target error) bool { return target == domain.ErrNotFound }
