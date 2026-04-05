// Package testdata provides test WASM binaries for the wasm package.
package testdata

// AddWasm is a minimal WASM module exporting an "add" function: (i32, i32) -> i32.
var AddWasm = []byte{
	0x00, 0x61, 0x73, 0x6d, // magic: \0asm
	0x01, 0x00, 0x00, 0x00, // version: 1

	// Type section (1 type)
	0x01, 0x07, // section id=1, size=7
	0x01,              // 1 type
	0x60,              // func
	0x02, 0x7f, 0x7f,  // 2 params: i32, i32
	0x01, 0x7f,        // 1 result: i32

	// Function section (1 function, type index 0)
	0x03, 0x02, // section id=3, size=2
	0x01, 0x00, // 1 function, type 0

	// Export section (1 export: "add")
	0x07, 0x07, // section id=7, size=7
	0x01,                    // 1 export
	0x03, 0x61, 0x64, 0x64, // name: "add"
	0x00, 0x00,              // kind: func, index: 0

	// Code section (1 function body)
	0x0a, 0x09, // section id=10, size=9
	0x01,       // 1 function body
	0x07,       // body size: 7
	0x00,       // 0 local declarations
	0x20, 0x00, // local.get 0
	0x20, 0x01, // local.get 1
	0x6a,       // i32.add
	0x0b,       // end
}

// InfiniteLoopWasm is a minimal WASM module exporting a "loop" function that
// runs forever (for timeout testing). It's a no-param no-result function with
// an infinite br loop.
var InfiniteLoopWasm = []byte{
	0x00, 0x61, 0x73, 0x6d, // magic
	0x01, 0x00, 0x00, 0x00, // version

	// Type section (1 type: () -> ())
	0x01, 0x04,
	0x01, 0x60, 0x00, 0x00,

	// Function section
	0x03, 0x02,
	0x01, 0x00,

	// Export section: "loop"
	0x07, 0x08,
	0x01,
	0x04, 0x6c, 0x6f, 0x6f, 0x70, // "loop"
	0x00, 0x00,

	// Code section: infinite loop via block+br
	0x0a, 0x09,
	0x01,       // 1 body
	0x07,       // body size
	0x00,       // 0 locals
	0x03, 0x40, // loop (block_type void)
	0x0c, 0x00, // br 0 (jump to loop start)
	0x0b,       // end loop
	0x0b,       // end func
}
