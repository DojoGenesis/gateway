//go:build tinygo

// Package main is a TinyGo WASM module that exposes DIP scoring as a WASI function.
//
// Build with: tinygo build -target=wasi -o dip-scorer.wasm -no-debug .
//
// Exported functions:
//   - compute_substance_score: accepts JSON ScoringRequest, returns JSON ScoringResult
//   - alloc: allocates WASM linear memory for the host to write input data
//   - free: frees previously allocated WASM memory
package main

import (
	"encoding/json"
	"unsafe"
)

//export alloc
func alloc(size uint32) *byte {
	buf := make([]byte, size)
	return &buf[0]
}

//export free
func free(ptr *byte, size uint32) {
	// TinyGo's GC will handle deallocation. This export exists so the
	// host runtime can signal that a buffer is no longer needed.
	_ = ptr
	_ = size
}

// readMemory reads size bytes starting at ptr from WASM linear memory.
func readMemory(ptr *byte, size uint32) []byte {
	return unsafe.Slice(ptr, size)
}

// writeMemory writes data into WASM linear memory starting at ptr.
func writeMemory(ptr *byte, data []byte) {
	dst := unsafe.Slice(ptr, len(data))
	copy(dst, data)
}

//export compute_substance_score
func computeSubstanceScore(inputPtr *byte, inputLen uint32) uint64 {
	// Read the JSON input from WASM memory.
	input := readMemory(inputPtr, inputLen)

	var req ScoringRequest
	if err := json.Unmarshal(input, &req); err != nil {
		// On parse error, return an error result.
		errResult := ScoringResult{
			AggregateScore: 0,
			DeviationState: "error",
			PerDimension:   nil,
		}
		return marshalAndReturn(errResult)
	}

	// Compute the score using the shared scoring logic (scorer.go).
	result := ComputeScore(req)

	return marshalAndReturn(result)
}

// marshalAndReturn serializes the result to JSON, allocates WASM memory for it,
// and returns a packed uint64 where the high 32 bits are the pointer and the
// low 32 bits are the length.
func marshalAndReturn(result ScoringResult) uint64 {
	output, err := json.Marshal(result)
	if err != nil {
		// Should never happen with our types, but handle gracefully.
		output = []byte(`{"aggregate_score":0,"deviation_state":"error","per_dimension":[]}`)
	}

	// Allocate output buffer in WASM memory.
	outPtr := alloc(uint32(len(output)))
	writeMemory(outPtr, output)

	// Pack pointer and length into a single uint64 for return.
	// High 32 bits = pointer, Low 32 bits = length.
	ptrVal := uint64(uintptr(unsafe.Pointer(outPtr)))
	lenVal := uint64(len(output))
	return (ptrVal << 32) | lenVal
}

func main() {
	// Required by TinyGo for WASI targets. The actual work is done
	// through the exported functions above.
}
