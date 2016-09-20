package fasthash

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func BenchmarkGet(b *testing.B) {
	list := []string{"node0", "node1"}
	data := []byte("112233445566")

	sj := New(list)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sj.Get(data)
	}
}

func TestGetError(t *testing.T) {
	assert := assert.New(t)

	data := []byte("112233445566")

	fh := New(nil)
	_, err := fh.Get(data)

	assert.NotNil(err)
}

func TestGetNormal(t *testing.T) {
	assert := assert.New(t)

	list := []string{"node0", "node1"}
	data := []byte("112233445566")

	fh := New(list)
	node, err := fh.Get(data)
	assert.Nil(err)
	assert.NotNil(node)

}
