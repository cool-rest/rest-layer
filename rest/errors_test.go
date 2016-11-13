package rest

import (
	"errors"
	"testing"

	"github.com/cool-rest/rest-layer/resource"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestNewError(t *testing.T) {
	assert.Equal(t, ErrClientClosedRequest, NewError(context.Canceled))
	assert.Equal(t, ErrGatewayTimeout, NewError(context.DeadlineExceeded))
	assert.Equal(t, ErrUnauthorized, NewError(resource.ErrUnauthorized))
	assert.Equal(t, ErrNotFound, NewError(resource.ErrNotFound))
	assert.Equal(t, ErrConflict, NewError(resource.ErrConflict))
	assert.Equal(t, ErrNotImplemented, NewError(resource.ErrNotImplemented))
	assert.Nil(t, NewError(nil))
	assert.Equal(t, &Error{520, "test", nil}, NewError(errors.New("test")))
	assert.Equal(t, ErrNotFound, NewError(ErrNotFound))
}

func TestError(t *testing.T) {
	e := &Error{123, "message", nil}
	assert.Equal(t, "message", e.Error())
}
