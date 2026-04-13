package server

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/DojoGenesis/gateway/server/database"
)

// ChunkText splits a document into overlapping chunks.
// chunkSize is in characters (not tokens). overlap is the number of characters
// shared between adjacent chunks. For CWD-scale documents, 2000-char chunks
// with 200-char overlap keeps context windows well under model limits.
//
// Split preference order:
//  1. Paragraph boundary (\n\n) within window
//  2. Sentence boundary (. or \n) within window
//  3. Hard split at chunkSize
func ChunkText(text string, chunkSize, overlap int) []string {
	if chunkSize <= 0 {
		chunkSize = 2000
	}
	if overlap < 0 || overlap >= chunkSize {
		overlap = 200
	}

	if len(text) == 0 {
		return nil
	}

	if len(text) <= chunkSize {
		return []string{text}
	}

	var chunks []string
	start := 0

	for start < len(text) {
		end := start + chunkSize
		if end >= len(text) {
			// Last chunk: take everything remaining
			chunk := strings.TrimSpace(text[start:])
			if chunk != "" {
				chunks = append(chunks, chunk)
			}
			break
		}

		// Look for the last paragraph break (\n\n) within the window.
		splitAt := -1
		if idx := strings.LastIndex(text[start:end], "\n\n"); idx != -1 {
			splitAt = start + idx + 2 // include both newlines, split after
		}

		// Fall back to last sentence boundary (period+space or bare newline).
		if splitAt == -1 {
			for i := end - 1; i > start; i-- {
				if text[i] == '\n' {
					splitAt = i + 1
					break
				}
				if text[i] == '.' && i+1 < len(text) && (text[i+1] == ' ' || text[i+1] == '\n') {
					splitAt = i + 1
					break
				}
			}
		}

		// Hard split if no boundary found.
		if splitAt == -1 || splitAt <= start {
			splitAt = end
		}

		chunk := strings.TrimSpace(text[start:splitAt])
		if chunk != "" {
			chunks = append(chunks, chunk)
		}

		// Next chunk starts with overlap
		next := splitAt - overlap
		if next <= start {
			next = splitAt // guard against infinite loop
		}
		start = next
	}

	return chunks
}

// ExtractText extracts plain text from uploaded file content.
// Supported types: .txt, .md (pass-through), .json (pretty-printed).
// PDFs are not supported without a native library — callers receive a
// friendly error suggesting they paste the text directly.
func ExtractText(content []byte, contentType string, filename string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))

	// Normalize content-type-based aliases
	switch {
	case ext == ".txt" || strings.Contains(contentType, "text/plain"):
		return string(content), nil

	case ext == ".md" || strings.Contains(contentType, "text/markdown"):
		return string(content), nil

	case ext == ".json" || strings.Contains(contentType, "application/json"):
		// Pretty-print JSON so chunks read more naturally
		var v interface{}
		if err := json.Unmarshal(content, &v); err != nil {
			// Malformed JSON — return raw string anyway
			return string(content), nil
		}
		pretty, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return string(content), nil
		}
		return string(pretty), nil

	case ext == ".pdf" || strings.Contains(contentType, "application/pdf"):
		return "", fmt.Errorf(
			"PDF text extraction is not supported in this version. " +
				"Please copy and paste the text from your PDF into a .txt file and upload that instead. " +
				"(PDF support via go-pdfium is planned for a future release)",
		)

	default:
		return "", fmt.Errorf(
			"unsupported file type %q (extension: %s): supported types are .txt, .md, .json",
			contentType, ext,
		)
	}
}

// ragChunkMeta holds the minimal document metadata needed to format RAG context.
// We look up filenames from the document store rather than storing them in each chunk.
type ragChunkMeta struct {
	chunk    *database.DocumentChunk
	filename string
}

// BuildRAGContext retrieves relevant chunks for a query and formats them as a
// system-prompt injection string. Returns "" if no relevant chunks are found.
//
// The formatted string is designed to be prepended to the system prompt so the
// model can cite specific documents when answering user questions.
func (s *Server) BuildRAGContext(ctx context.Context, userID string, query string, maxChunks int) (string, error) {
	if s.authDB == nil {
		return "", nil
	}

	if maxChunks <= 0 {
		maxChunks = 5
	}

	adapter := database.NewLocalAdapter(s.authDB)

	chunks, err := adapter.SearchDocumentChunks(ctx, userID, query, maxChunks)
	if err != nil {
		return "", fmt.Errorf("rag: chunk search failed: %w", err)
	}

	if len(chunks) == 0 {
		return "", nil
	}

	// Resolve filenames — batch by unique document IDs to minimise queries.
	filenameByDocID := make(map[string]string, len(chunks))
	for _, c := range chunks {
		if _, seen := filenameByDocID[c.DocumentID]; !seen {
			doc, err := adapter.GetDocument(ctx, c.DocumentID)
			if err == nil {
				filenameByDocID[c.DocumentID] = doc.Filename
			} else {
				filenameByDocID[c.DocumentID] = c.DocumentID // fallback to ID
			}
		}
	}

	var sb strings.Builder
	sb.WriteString("The following context from uploaded documents may be relevant:\n\n")

	for _, c := range chunks {
		filename := filenameByDocID[c.DocumentID]
		fmt.Fprintf(&sb, "--- From: %s (chunk %d) ---\n%s\n\n", filename, c.ChunkIndex+1, c.Content)
	}

	sb.WriteString("Use this context to inform your response. Cite the document name when referencing specific information.")

	return sb.String(), nil
}
