package resetter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testObject struct {
	id int
}

func (o *testObject) Reset() {}

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		creator     func() *testObject
		expectError bool
	}{
		{
			name:        "nil creator returns error",
			creator:     nil,
			expectError: true,
		},
		{
			name: "valid creator succeeds",
			creator: func() *testObject {
				return &testObject{}
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool, err := New(tt.creator)

			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, pool)
			} else {
				require.NoError(t, err)
				require.NotNil(t, pool)
			}
		})
	}
}

func TestPool_Get_EmptyPoolCreatesNew(t *testing.T) {
	var createCount int

	pool, err := New(func() *testObject {
		createCount++
		return &testObject{id: createCount}
	})
	require.NoError(t, err)

	obj := pool.Get()

	require.NotNil(t, obj)
	assert.Equal(t, 1, createCount)
	assert.Equal(t, 1, obj.id)
}

func TestPool_Get_MultipleEmptyPoolCreatesNew(t *testing.T) {
	var createCount int

	pool, err := New(func() *testObject {
		createCount++
		return &testObject{id: createCount}
	})
	require.NoError(t, err)

	first := pool.Get()
	second := pool.Get()

	assert.Equal(t, 2, createCount)
	assert.NotSame(t, first, second)
}

func TestPool_Get_ReusesPutObjectInsteadOfCreatingNew(t *testing.T) {
	var createCount int

	pool, err := New(func() *testObject {
		createCount++
		return &testObject{id: createCount}
	})
	require.NoError(t, err)

	original := &testObject{id: 999}
	pool.Put(original)

	obj := pool.Get()

	assert.Same(t, original, obj)
	assert.Equal(t, 0, createCount, "creator must not be invoked while pool has objects available")
}

func TestPool_Get_LIFOOrder(t *testing.T) {
	pool, err := New(func() *testObject {
		return &testObject{}
	})
	require.NoError(t, err)

	obj1 := &testObject{id: 1}
	obj2 := &testObject{id: 2}
	obj3 := &testObject{id: 3}

	pool.Put(obj1)
	pool.Put(obj2)
	pool.Put(obj3)

	assert.Same(t, obj3, pool.Get())
	assert.Same(t, obj2, pool.Get())
	assert.Same(t, obj1, pool.Get())
}

func TestPool_Put_AddsObjectToInternalSlice(t *testing.T) {
	pool, err := New(func() *testObject {
		return &testObject{}
	})
	require.NoError(t, err)

	obj := &testObject{id: 1}
	pool.Put(obj)

	require.Len(t, pool.objects, 1)
	assert.Same(t, obj, pool.objects[0])
}

func TestPool_Put_NilObjectIsIgnored(t *testing.T) {
	pool, err := New(func() *testObject {
		return &testObject{}
	})
	require.NoError(t, err)

	var nilObj *testObject
	pool.Put(nilObj)

	assert.Empty(t, pool.objects, "nil object must not be stored in the pool")
}

func TestPool_Put_MultipleObjectsAccumulate(t *testing.T) {
	pool, err := New(func() *testObject {
		return &testObject{}
	})
	require.NoError(t, err)

	pool.Put(&testObject{id: 1})
	pool.Put(&testObject{id: 2})
	pool.Put(&testObject{id: 3})

	assert.Len(t, pool.objects, 3)
}
