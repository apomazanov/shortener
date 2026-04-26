package repository

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

/* -------------------------------------------------------------------------- */
func TestRepoMemory_Add(t *testing.T) {

	repo := NewInMemoryRepo()

	type expected struct {
		value string
		ok bool
	}

	tests := []struct {
		name string
		expected expected
	} {
		{
			name: "first add",
			expected: expected{
				value: "short1",
				ok: true, // always true while in-memory storage used
			},
		},
		{
			name: "second add",
			expected: expected{
				value: "short2",
				ok: true,
			},
		},
		{
			name: "third add",
			expected: expected{
				value: "short3",
				ok: true,
			},
		},
	}

	for i, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, ok := repo.Add(fmt.Sprintf("abracadabra%d", i))
			assert.Equal(t, test.expected.ok, ok) 
			assert.Equal(t, test.expected.value, result)
		})
	}
}

/* -------------------------------------------------------------------------- */
func TestRepoMemory_Get(t *testing.T) {

	repo := NewInMemoryRepo()
	repo.data["short1"] = "abracadabra1"
	repo.data["short2"] = "abracadabra2"
	repo.data["short3"] = "abracadabra3"

	t.Run("success", func(t *testing.T){
		result, ok := repo.Get("short1")
		assert.Equal(t, "abracadabra1", result)
		assert.True(t, ok)
	})

	t.Run("not found", func(t *testing.T){
		result, ok := repo.Get("short4")
		assert.Equal(t, "", result)
		assert.False(t, ok)
	})
}