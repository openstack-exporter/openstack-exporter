package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemoveElements(t *testing.T) {
	result := RemoveElements([]string{"a", "b", "c", "b"}, []string{"b"})
	assert.Equal(t, []string{"a", "c"}, result)
}

func TestUniqueElements(t *testing.T) {
	result := UniqueElements([]string{"a", "b", "a", "c", "b"})
	assert.Equal(t, []string{"a", "b", "c"}, result)
}
