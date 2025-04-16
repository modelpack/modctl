package interceptor

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalcCrc32_NormalExecution_ReturnsCorrectCrc32(t *testing.T) {
	data := []byte("hello world")
	expectedCrc32 := []uint32{0xc99465aa}

	crc32Results, err := calcCrc32(context.Background(), bytes.NewReader(data), int64(len(data)))
	assert.NoError(t, err)
	assert.Equal(t, expectedCrc32, crc32Results)
}

func TestCalcCrc32_EmptyInput_ReturnsEmptySlice(t *testing.T) {
	crc32Results, err := calcCrc32(context.Background(), bytes.NewReader([]byte{}), 10)
	assert.NoError(t, err)
	assert.NotEmpty(t, crc32Results)
	assert.Equal(t, uint32(0x0), crc32Results[0])
}
