package extensions

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTerminalApplyError(t *testing.T) {
	cause := errors.New("pull image: manifest unknown")
	err := NewTerminalApplyError(cause)

	require.Error(t, err)
	assert.EqualError(t, err, cause.Error())
	assert.ErrorIs(t, err, cause)
	assert.True(t, IsTerminalApplyError(err))
	assert.True(t, IsTerminalApplyError(fmt.Errorf("apply run: %w", err)))
	assert.False(t, IsTerminalApplyError(cause))
	assert.Same(t, err, NewTerminalApplyError(err))
	assert.NoError(t, NewTerminalApplyError(nil))
}
