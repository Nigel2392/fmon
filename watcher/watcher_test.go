package watcher_test

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"unsafe"
)

// DirSize1 uses the standard library's filepath.Walk
func DirSize1(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}

// DirSize2 uses custom recursive os.ReadDir & os.Lstat
func DirSize2(path string) (int64, error) {
	var size int64
	var calculateSize func(string) error
	calculateSize = func(p string) error {
		fileInfo, err := os.Lstat(p)
		if err != nil {
			return err
		}

		if fileInfo.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		if fileInfo.IsDir() {
			entries, err := os.ReadDir(p)
			if err != nil {
				return err
			}
			for _, entry := range entries {
				if err := calculateSize(filepath.Join(p, entry.Name())); err != nil {
					return err
				}
			}
		} else {
			size += fileInfo.Size()
		}
		return nil
	}

	if err := calculateSize(path); err != nil {
		return 0, err
	}

	return size, nil
}

func DirSize3(path string) (int64, error) {
	var size int64

	var worker func(string) error
	worker = func(p string) error {
		entries, err := os.ReadDir(p)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			if entry.Type()&os.ModeSymlink != 0 {
				continue
			}

			if entry.IsDir() {
				if err := worker(filepath.Join(p, entry.Name())); err != nil {
					return err
				}
			} else {
				// Reuses metadata already pulled by ReadDir where possible
				info, err := entry.Info()
				if err != nil {
					return err
				}
				size += info.Size()
			}
		}
		return nil
	}

	// Just check the root once to ensure it's a directory
	rootInfo, err := os.Lstat(path)
	if err != nil {
		return 0, err
	}
	if !rootInfo.IsDir() {
		return rootInfo.Size(), nil
	}

	err = worker(path)
	return size, err
}

func DirSize4(path string) (int64, error) {
	// Clean the path once up front
	if len(path) == 0 {
		return 0, nil
	}

	// Ensure standard Windows backslash formatting without trailing slash
	if path[len(path)-1] == '/' || path[len(path)-1] == '\\' {
		path = path[:len(path)-1]
	}

	var size int64

	// Define a flat worker that accepts a pre-allocated string builder
	// or uses optimized primitive concatenation.
	var worker func(string) error
	worker = func(currentPath string) error {
		entries, err := os.ReadDir(currentPath)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			if entry.Type()&os.ModeSymlink != 0 {
				continue
			}

			// Manual concatenation is faster than filepath.Join
			// because it bypasses generic path cleaning logic.
			fullPath := currentPath + "\\" + entry.Name()

			if entry.IsDir() {
				if err := worker(fullPath); err != nil {
					return err
				}
			} else {
				info, err := entry.Info()
				if err != nil {
					return err
				}
				size += info.Size()
			}
		}
		return nil
	}

	rootInfo, err := os.Lstat(path)
	if err != nil {
		return 0, err
	}
	if !rootInfo.IsDir() {
		return rootInfo.Size(), nil
	}

	err = worker(path)
	return size, err
}

func DirSize5(path string) (int64, error) {
	var size int64
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Capture the first error across goroutines safely
	var errOnce sync.Once
	var globalErr error
	setErr := func(e error) {
		errOnce.Do(func() {
			globalErr = e
		})
	}

	var worker func(string)
	worker = func(currentPath string) {
		defer wg.Done()

		entries, err := os.ReadDir(currentPath)
		if err != nil {
			setErr(err)
			return
		}

		var localSize int64

		for _, entry := range entries {
			if entry.Type()&os.ModeSymlink != 0 {
				continue
			}

			if entry.IsDir() {
				// Launch subdirectories in parallel
				wg.Add(1)
				go worker(filepath.Join(currentPath, entry.Name()))
			} else {
				// No path allocation needed for files!
				info, err := entry.Info()
				if err != nil {
					setErr(err)
					return
				}
				localSize += info.Size()
			}
		}

		// Bulk add to the global counter to minimize mutex contention
		if localSize > 0 {
			mu.Lock()
			size += localSize
			mu.Unlock()
		}
	}

	rootInfo, err := os.Lstat(path)
	if err != nil {
		return 0, err
	}
	if !rootInfo.IsDir() {
		return rootInfo.Size(), nil
	}

	wg.Add(1)
	go worker(path)
	wg.Wait()

	if globalErr != nil {
		return 0, globalErr
	}

	return size, nil
}

func DirSize6(path string) (int64, error) {
	rootInfo, err := os.Lstat(path)
	if err != nil {
		return 0, err
	}
	if !rootInfo.IsDir() {
		return rootInfo.Size(), nil
	}

	var size int64
	err = dirSize6Worker(path, &size)
	return size, err
}

// Moving this outside avoids the memory overhead of a closure.
// Passing a pointer to size keeps memory mutations on the stack.
func dirSize6Worker(currentPath string, totalSize *int64) error {
	entries, err := os.ReadDir(currentPath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}

		if entry.IsDir() {
			// Only allocate a new path string when stepping into a directory.
			// Direct string concatenation is faster than filepath.Join here
			// because we already know our baseline formatting.
			subDir := currentPath + "\\" + entry.Name()
			if err := dirSize6Worker(subDir, totalSize); err != nil {
				return err
			}
		} else {
			// Zero path allocation for files. Uses pre-fetched descriptor info.
			info, err := entry.Info()
			if err != nil {
				return err
			}
			*totalSize += info.Size()
		}
	}
	return nil
}

func DirSize7(path string) (int64, error) {
	rootInfo, err := os.Lstat(path)
	if err != nil {
		return 0, err
	}
	if !rootInfo.IsDir() {
		return rootInfo.Size(), nil
	}

	var size int64

	// Pre-allocate a 1KB buffer. This is large enough to handle incredibly
	// deep Windows paths without ever triggering a slice capacity expansion.
	buf := make([]byte, 0, 1024)
	buf = append(buf, path...)

	err = dirSize7Worker(&buf, &size)
	return size, err
}

func dirSize7Worker(buf *[]byte, totalSize *int64) error {
	// Zero-allocation byte-to-string conversion (Requires Go 1.20+)
	// This tells Go: "Treat this byte slice as a string, don't copy the memory."
	// It is completely safe here because os.ReadDir only reads the string synchronously.
	currentPath := unsafe.String(unsafe.SliceData(*buf), len(*buf))

	entries, err := os.ReadDir(currentPath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}

		if entry.IsDir() {
			// Save the current path length
			origLen := len(*buf)

			// Append the separator and the new directory name directly to the buffer
			*buf = append(*buf, '\\')
			*buf = append(*buf, entry.Name()...)

			if err := dirSize7Worker(buf, totalSize); err != nil {
				return err
			}

			// BACKTRACK: Instantly reset the buffer length to drop the child directory,
			// allowing the next loop iteration to overwrite that memory.
			*buf = (*buf)[:origLen]
		} else {
			info, err := entry.Info()
			if err != nil {
				return err
			}
			*totalSize += info.Size()
		}
	}
	return nil
}

func BenchmarkDirSize(b *testing.B) {
	b.StopTimer()
	targetPath := filepath.Clean(`C:\ai`)

	var size, err = DirSize1(targetPath)
	if err != nil {
		b.Fatal(err)
	}

	var funcs = []func(string) (int64, error){
		DirSize1,
		DirSize2,
		DirSize3,
		DirSize4,
		DirSize5,
		DirSize6,
		DirSize7,
	}

	for idx, fn := range funcs {
		b.Run(fmt.Sprintf("BenchmarkDirSize%d", idx+1), func(b *testing.B) {

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				s, err := fn(targetPath)
				if err != nil {
					b.Fatal(err)
				}

				if s != size {
					b.Fatalf("Size mismatch: %d != %d", s, size)
				}
			}
		})
	}
}
