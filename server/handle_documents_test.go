package server

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── ChunkText tests ─────────────────────────────────────────────────────────

func TestChunkText_LargeInput(t *testing.T) {
	// Build a 5000-char text of repeated sentences
	sentence := "The quick brown fox jumps over the lazy dog. "
	text := strings.Repeat(sentence, 5000/len(sentence)+1)
	text = text[:5000]

	chunks := ChunkText(text, 2000, 200)

	// With 5000 chars, 2000 chunk size, 200 overlap we expect 3 chunks
	// (exact count depends on boundary alignment, but must be > 1)
	assert.Greater(t, len(chunks), 1, "large text should produce multiple chunks")

	// Verify no chunk exceeds chunkSize significantly
	for i, ch := range chunks {
		assert.LessOrEqual(t, len(ch), 2000+50, "chunk %d length should be near chunkSize", i)
	}

	// Verify overlap: last overlap chars of chunk[i] should appear at start of chunk[i+1]
	// (approximate check — boundary snapping may shift the overlap slightly)
	if len(chunks) >= 2 {
		end0 := chunks[0]
		start1 := chunks[1]
		// At least some suffix of chunk 0 should be a prefix-region of chunk 1
		overlap := chunks[0][len(end0)-100:]
		assert.True(t, strings.Contains(start1, overlap[:20]),
			"chunk 1 should contain some content from the end of chunk 0 (overlap)")
	}
}

func TestChunkText_SmallInput(t *testing.T) {
	text := "This is a short document."
	chunks := ChunkText(text, 2000, 200)
	require.Len(t, chunks, 1, "text smaller than chunkSize should produce exactly one chunk")
	assert.Equal(t, text, chunks[0])
}

func TestChunkText_EmptyInput(t *testing.T) {
	chunks := ChunkText("", 2000, 200)
	assert.Nil(t, chunks, "empty text should return nil")
}

func TestChunkText_ParagraphBoundary(t *testing.T) {
	// Build text with a paragraph break near the middle.
	// First paragraph: ~1000 chars, second paragraph: ~1000 chars.
	para1 := strings.Repeat("A sentence here. ", 60) // ~1020 chars
	para2 := strings.Repeat("Another sentence. ", 60) // ~1080 chars
	text := para1 + "\n\n" + para2

	chunks := ChunkText(text, 1200, 100)
	require.Greater(t, len(chunks), 1, "text should be split into multiple chunks")

	// The split should prefer the paragraph boundary — chunk 0 should end
	// around the paragraph break and NOT contain a large slab of para2.
	// Check that chunk 0 contains para1 content and minimal para2 content.
	firstChunk := chunks[0]
	assert.Contains(t, firstChunk, "A sentence here.", "first chunk should contain paragraph 1 content")
	// The paragraph boundary (\n\n) should cause a split so chunk 0 ends near it.
	// chunk 0 should not contain more than overlap-worth of para2.
	para2Start := "Another sentence."
	idx := strings.Index(firstChunk, para2Start)
	if idx != -1 {
		// If para2 content leaked into chunk 0 via overlap, it should be at most
		// the overlap region (last 100 chars of chunk 0).
		assert.GreaterOrEqual(t, idx, len(firstChunk)-200,
			"para2 content in chunk 0 should only appear in the overlap tail")
	}
}

func TestChunkText_DefaultParameters(t *testing.T) {
	// Passing 0/negative values should use sensible defaults (2000, 200)
	text := strings.Repeat("word ", 1000) // 5000 chars
	chunks := ChunkText(text, 0, -1)
	assert.Greater(t, len(chunks), 1, "zero chunkSize should use default and produce multiple chunks")
}

func TestChunkText_ExactlyOneChunkSize(t *testing.T) {
	text := strings.Repeat("x", 2000)
	chunks := ChunkText(text, 2000, 200)
	require.Len(t, chunks, 1)
	assert.Equal(t, text, chunks[0])
}

// ─── ExtractText tests ────────────────────────────────────────────────────────

func TestExtractText_Plaintext(t *testing.T) {
	content := []byte("Hello, world! This is plain text.")
	result, err := ExtractText(content, "text/plain", "notes.txt")
	require.NoError(t, err)
	assert.Equal(t, string(content), result)
}

func TestExtractText_Markdown(t *testing.T) {
	content := []byte("# Heading\n\nSome **bold** text.")
	result, err := ExtractText(content, "text/markdown", "doc.md")
	require.NoError(t, err)
	assert.Equal(t, string(content), result)
}

func TestExtractText_MarkdownByExtension(t *testing.T) {
	// Extension takes precedence over content type
	content := []byte("# Atlas Notes\n\nPolicy data.")
	result, err := ExtractText(content, "application/octet-stream", "readme.md")
	require.NoError(t, err)
	assert.Equal(t, string(content), result)
}

func TestExtractText_JSON(t *testing.T) {
	content := []byte(`{"key":"value","count":42}`)
	result, err := ExtractText(content, "application/json", "data.json")
	require.NoError(t, err)
	// Should be pretty-printed
	assert.Contains(t, result, "\"key\"")
	assert.Contains(t, result, "\"value\"")
	assert.True(t, strings.Contains(result, "\n"), "JSON should be pretty-printed with newlines")
}

func TestExtractText_MalformedJSON(t *testing.T) {
	// Malformed JSON should fall through and return raw content without error
	content := []byte(`{not valid json`)
	result, err := ExtractText(content, "application/json", "bad.json")
	require.NoError(t, err)
	assert.Equal(t, string(content), result)
}

func TestExtractText_PDF(t *testing.T) {
	content := []byte("%PDF-1.4 fake content")
	_, err := ExtractText(content, "application/pdf", "report.pdf")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PDF text extraction is not supported")
	assert.Contains(t, err.Error(), ".txt file")
}

func TestExtractText_UnsupportedType(t *testing.T) {
	content := []byte{0x50, 0x4B, 0x03, 0x04} // ZIP magic bytes
	_, err := ExtractText(content, "application/zip", "archive.zip")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported file type")
}

func TestExtractText_TxtByExtension(t *testing.T) {
	// .txt with wrong content-type should still work
	content := []byte("Grant narrative draft.")
	result, err := ExtractText(content, "application/octet-stream", "grant.txt")
	require.NoError(t, err)
	assert.Equal(t, string(content), result)
}
