package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsMicroversionAtLeast(t *testing.T) {
	tests := []struct {
		name     string
		current  string
		required string
		want     bool
	}{
		{name: "same version", current: "2.46", required: "2.46", want: true},
		{name: "higher minor", current: "2.87", required: "2.46", want: true},
		{name: "lower minor", current: "2.32", required: "2.33", want: false},
		{name: "higher major", current: "3.1", required: "2.99", want: true},
		{name: "lower major", current: "1.99", required: "2.1", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IsMicroversionAtLeast(tt.current, tt.required)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestIsMicroversionAtLeastInvalidVersion(t *testing.T) {
	_, err := IsMicroversionAtLeast("invalid", "2.46")
	require.Error(t, err)

	_, err = IsMicroversionAtLeast("2.46", "invalid")
	require.Error(t, err)
}

