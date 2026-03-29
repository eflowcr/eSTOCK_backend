package tools

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStrPtr(t *testing.T) {
	s := "hello"
	ptr := StrPtr(s)
	require.NotNil(t, ptr)
	assert.Equal(t, s, *ptr)
	// Different pointer
	assert.NotSame(t, &s, ptr)
}

func TestStringToInt(t *testing.T) {
	t.Run("empty string returns 0", func(t *testing.T) {
		v, err := StringToInt("")
		require.NoError(t, err)
		assert.Equal(t, 0, v)
	})

	t.Run("valid integer", func(t *testing.T) {
		v, err := StringToInt("42")
		require.NoError(t, err)
		assert.Equal(t, 42, v)
	})

	t.Run("negative integer", func(t *testing.T) {
		v, err := StringToInt("-7")
		require.NoError(t, err)
		assert.Equal(t, -7, v)
	})

	t.Run("invalid string returns error", func(t *testing.T) {
		_, err := StringToInt("abc")
		assert.Error(t, err)
	})
}

func TestParseDate(t *testing.T) {
	t.Run("empty string returns nil", func(t *testing.T) {
		assert.Nil(t, ParseDate(""))
	})

	t.Run("unrecognized format returns nil", func(t *testing.T) {
		assert.Nil(t, ParseDate("not-a-date"))
	})

	cases := []struct {
		input    string
		expected time.Time
	}{
		{"2024-03-15", time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)},
		{"15-03-2024", time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)},
		{"2024/03/15", time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)},
		{"15/03/2024", time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)},
		{"2024.03.15", time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)},
		{"15.03.2024", time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)},
		{"2024-3-5", time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC)},
		{"5-3-2024", time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC)},
		{"2024/3/5", time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC)},
		{"5/3/2024", time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC)},
		{"2024.3.5", time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC)},
		{"5.3.2024", time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC)},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := ParseDate(tc.input)
			require.NotNil(t, got)
			assert.Equal(t, tc.expected, *got)
		})
	}
}
