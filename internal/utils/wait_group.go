package utils

import (
	"reflect"
	"sync"
	"unsafe"
)

// GetWaitGroupCount retrieves the current counter value from a sync.WaitGroup
// NOTE: This is a hack and should only be used for debugging purposes
func GetWaitGroupCount(wg *sync.WaitGroup) int {
	// Use reflection to access the unexported counter field
	wgValue := reflect.ValueOf(wg).Elem()
	counterValue := wgValue.FieldByName("counter")
	
	// Access unexported field
	if !counterValue.IsValid() {
		return -1 // Unable to access counter
	}
	
	// Get the counter value using unsafe
	counterPtr := unsafe.Pointer(counterValue.UnsafeAddr())
	return int(*(*int32)(counterPtr))
}
