package memory

import (
	"encoding/binary"
	"fmt"
	"math"
)

// ExpectedEmbeddingDim is the expected dimension for embeddings.
const ExpectedEmbeddingDim = 768

func validateEmbedding(embedding []float32) error {
	if len(embedding) != ExpectedEmbeddingDim {
		return fmt.Errorf("invalid embedding dimension: expected %d, got %d", ExpectedEmbeddingDim, len(embedding))
	}
	return nil
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}

	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct float64
	var normA float64
	var normB float64

	for i := 0; i < len(a); i++ {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0.0 || normB == 0.0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

func serializeEmbedding(embedding []float32) ([]byte, error) {
	if len(embedding) == 0 {
		return nil, fmt.Errorf("embedding cannot be empty")
	}

	buf := make([]byte, len(embedding)*4)
	for i, val := range embedding {
		binary.LittleEndian.PutUint32(buf[i*4:(i+1)*4], math.Float32bits(val))
	}

	return buf, nil
}

func deserializeEmbedding(data []byte) ([]float32, error) {
	if len(data) == 0 {
		return nil, nil
	}

	if len(data)%4 != 0 {
		return nil, fmt.Errorf("invalid embedding data: length must be multiple of 4, got %d", len(data))
	}

	embedding := make([]float32, len(data)/4)
	for i := 0; i < len(embedding); i++ {
		bits := binary.LittleEndian.Uint32(data[i*4 : (i+1)*4])
		embedding[i] = math.Float32frombits(bits)
	}

	return embedding, nil
}
