//go:build !tinygo

package main

// stub_main.go satisfies the Go toolchain's requirement for a main() function
// when building with standard Go (not TinyGo). The real entry point and WASM
// exports live in main.go, which is only included under the tinygo build tag.
func main() {}
