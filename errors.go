package main

import "github.com/rzh0504/qmdr/internal/apperr"

func friendlyError(err error) string { return apperr.Friendly(err) }
