package services

import (
	"fmt"
	"math/rand"
	"time"
	"unsafe"
)

// Random generation for tests

// DataRandCS is charset for randomizing
const DataRandCS = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ-"

var src = rand.NewSource(time.Now().UnixNano())

func randomByteSlice(size int, prefix string, charset string) []byte {
	csLen := len(charset)
	prefLen := len(prefix)
	result := make([]byte, size)
	copy(result, prefix)
	for i := prefLen; i < size; i++ {
		result[i] = charset[src.Int63()%int64(csLen)]
	}
	return result
}

// RandomString generates random string
func RandomString(size int, prefix string, charset ...string) string {
	cs := DataRandCS
	if len(charset) > 0 {
		cs = charset[0]
	}
	result := randomByteSlice(size, prefix, cs)
	return *(*string)(unsafe.Pointer(&result)) // faster way to convert big slice to string
}

func WaitForSpecificOrError(f func() (bool, error), maxAttempts int, waitInterval time.Duration) error {
	for i := 0; i < maxAttempts; i++ {
		stop, err := f()
		if err != nil {
			return err
		}
		if stop {
			return nil
		}
		time.Sleep(waitInterval)
	}
	return fmt.Errorf("Maximum number of retries (%d) exceeded", maxAttempts)
}
