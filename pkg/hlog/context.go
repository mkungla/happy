// Copyright 2022 Marko Kungla
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package hlog

import "context"

type contextKey struct{}

// NewContext returns a context that contains the given Logger.
// Use FromContext to retrieve the Logger.
func NewContext(ctx context.Context, l *Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, l)
}

// FromContext returns the Logger stored in ctx by NewContext, or the default
// Logger if there is none.
func FromContext(ctx context.Context) *Logger {
	if l, ok := ctx.Value(contextKey{}).(*Logger); ok {
		return l
	}
	return Default()
}

// Ctx retrieves a Logger from the given context using FromContext. Then it adds
// the given context to the Logger using WithContext and returns the result.
func Ctx(ctx context.Context) *Logger {
	return FromContext(ctx).WithContext(ctx)
}
