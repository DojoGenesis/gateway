package memory

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateEmbedding(t *testing.T) {
	t.Run("valid embedding", func(t *testing.T) {
		embedding := make([]float32, ExpectedEmbeddingDim)
		err := validateEmbedding(embedding)
		assert.NoError(t, err)
	})

	t.Run("wrong dimension", func(t *testing.T) {
		embedding := make([]float32, 100)
		err := validateEmbedding(embedding)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid embedding dimension")
	})

	t.Run("empty embedding", func(t *testing.T) {
		err := validateEmbedding([]float32{})
		assert.Error(t, err)
	})
}

func TestCosineSimilarity(t *testing.T) {
	t.Run("identical vectors", func(t *testing.T) {
		a := []float32{1.0, 0.0, 0.0}
		sim := cosineSimilarity(a, a)
		assert.InDelta(t, 1.0, sim, 0.001)
	})

	t.Run("orthogonal vectors", func(t *testing.T) {
		a := []float32{1.0, 0.0, 0.0}
		b := []float32{0.0, 1.0, 0.0}
		sim := cosineSimilarity(a, b)
		assert.InDelta(t, 0.0, sim, 0.001)
	})

	t.Run("opposite vectors", func(t *testing.T) {
		a := []float32{1.0, 0.0}
		b := []float32{-1.0, 0.0}
		sim := cosineSimilarity(a, b)
		assert.InDelta(t, -1.0, sim, 0.001)
	})

	t.Run("empty vectors", func(t *testing.T) {
		sim := cosineSimilarity([]float32{}, []float32{})
		assert.Equal(t, 0.0, sim)
	})

	t.Run("mismatched lengths", func(t *testing.T) {
		a := []float32{1.0, 2.0}
		b := []float32{1.0}
		sim := cosineSimilarity(a, b)
		assert.Equal(t, 0.0, sim)
	})

	t.Run("zero vector", func(t *testing.T) {
		a := []float32{0.0, 0.0, 0.0}
		b := []float32{1.0, 2.0, 3.0}
		sim := cosineSimilarity(a, b)
		assert.Equal(t, 0.0, sim)
	})

	t.Run("real-world sized vectors", func(t *testing.T) {
		a := make([]float32, 768)
		b := make([]float32, 768)
		for i := range a {
			a[i] = float32(i) / 768.0
			b[i] = float32(i) / 768.0
		}
		sim := cosineSimilarity(a, b)
		assert.InDelta(t, 1.0, sim, 0.001)
	})

	t.Run("similarity is between -1 and 1", func(t *testing.T) {
		a := []float32{0.5, 0.3, 0.8, 0.1}
		b := []float32{0.2, 0.9, 0.4, 0.6}
		sim := cosineSimilarity(a, b)
		assert.True(t, sim >= -1.0 && sim <= 1.0)
	})
}

func TestSerializeDeserializeEmbedding(t *testing.T) {
	t.Run("round trip", func(t *testing.T) {
		original := []float32{1.0, 2.5, -3.14, 0.0, 100.0}
		data, err := serializeEmbedding(original)
		require.NoError(t, err)
		assert.Equal(t, len(original)*4, len(data))

		restored, err := deserializeEmbedding(data)
		require.NoError(t, err)
		assert.Equal(t, len(original), len(restored))

		for i := range original {
			assert.InDelta(t, float64(original[i]), float64(restored[i]), 0.0001)
		}
	})

	t.Run("serialize empty returns error", func(t *testing.T) {
		_, err := serializeEmbedding([]float32{})
		assert.Error(t, err)
	})

	t.Run("deserialize empty returns nil", func(t *testing.T) {
		result, err := deserializeEmbedding([]byte{})
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("deserialize invalid length", func(t *testing.T) {
		_, err := deserializeEmbedding([]byte{0x01, 0x02, 0x03})
		assert.Error(t, err)
	})

	t.Run("large embedding round trip", func(t *testing.T) {
		original := make([]float32, 768)
		for i := range original {
			original[i] = float32(math.Sin(float64(i)))
		}

		data, err := serializeEmbedding(original)
		require.NoError(t, err)

		restored, err := deserializeEmbedding(data)
		require.NoError(t, err)
		assert.Equal(t, len(original), len(restored))

		for i := range original {
			assert.InDelta(t, float64(original[i]), float64(restored[i]), 0.0001)
		}
	})

	t.Run("special float values", func(t *testing.T) {
		original := []float32{float32(math.Inf(1)), float32(math.Inf(-1)), 0.0}
		data, err := serializeEmbedding(original)
		require.NoError(t, err)

		restored, err := deserializeEmbedding(data)
		require.NoError(t, err)
		assert.True(t, math.IsInf(float64(restored[0]), 1))
		assert.True(t, math.IsInf(float64(restored[1]), -1))
	})
}
