package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntToFloat64(t *testing.T) {
	assert.Equal(t, float64(0), IntToFloat64(0))
	assert.Equal(t, float64(42), IntToFloat64(42))
	assert.Equal(t, float64(-5), IntToFloat64(-5))
}

func TestIntToPtr(t *testing.T) {
	ptr := IntToPtr(7)
	require.NotNil(t, ptr)
	assert.Equal(t, 7, *ptr)

	zero := IntToPtr(0)
	require.NotNil(t, zero)
	assert.Equal(t, 0, *zero)
}
