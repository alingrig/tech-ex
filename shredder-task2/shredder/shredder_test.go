package shredder

import (
	"bytes"
	"fmt"
	"os"
	"testing"
)

func createTempFile(t *testing.T, size int64) string {
	t.Helper()
	f, err := os.CreateTemp("", "shred-test-*")
	if err != nil {
		t.Fatal(err)
	}
	path := f.Name()
	if err := f.Truncate(size); err != nil {
		f.Close()
		os.Remove(path)
		t.Fatal(err)
	}
	f.Close()
	return path
}

// test random content return the correct amount of bytes
func TestGenRandomContent_ReturnsCorrectLength(t *testing.T) {
	for _, n := range []int{0, 1, 512, 4096, 1024 * 1024} {
		n := n
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			b, err := genRandomContent(n)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(b) != n {
				t.Errorf("got len=%d, want %d", len(b), n)
			}
		})
	}
}

// test the random content is indeed random
func TestGenRandomContent_IsNonDeterministic(t *testing.T) {
	a, err := genRandomContent(64)
	if err != nil {
		t.Fatal(err)
	}
	b, err := genRandomContent(64)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(a, b) {
		t.Error("two independent 64-byte buffers are identical – very suspicious")
	}
}

// shredChunk should write the whole payload correctly
func TestShredChunk_WritesPayloadCorrectly(t *testing.T) {
	path := createTempFile(t, 4096)
	defer os.Remove(path)

	f, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	payload := bytes.Repeat([]byte{0xAB}, 1024)
	if err := shredChunk(f, 0, 4096, payload); err != nil {
		t.Fatalf("shredChunk error: %v", err)
	}

	got := make([]byte, 1024)
	if _, err := f.ReadAt(got, 0); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Error("read-back bytes do not match written payload")
	}
}

// shredChunk should write at the correct offset.
func TestShredChunk_WritesAtMidFileOffset(t *testing.T) {
	const fileSize = int64(4096)
	path := createTempFile(t, fileSize)
	defer os.Remove(path)

	f, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	const offset = int64(1024)
	payload := bytes.Repeat([]byte{0xCD}, 1024)
	if err := shredChunk(f, offset, fileSize, payload); err != nil {
		t.Fatalf("shredChunk error: %v", err)
	}

	got := make([]byte, 1024)
	if _, err := f.ReadAt(got, offset); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("mid-file write mismatch at offset %d", offset)
	}
}

// shredChunk must never extend the file beyond its original size.
func TestShredChunk_DoesNotGrowFile(t *testing.T) {
	const fileSize = int64(512)
	path := createTempFile(t, fileSize)
	defer os.Remove(path)

	f, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	// Content is intentionally 4× the file size.
	content := bytes.Repeat([]byte{0xFF}, 2048)
	if err := shredChunk(f, 0, fileSize, content); err != nil {
		t.Fatalf("shredChunk error: %v", err)
	}

	stat, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if stat.Size() != fileSize {
		t.Errorf("file size changed: got %d, want %d", stat.Size(), fileSize)
	}
}

// For each representative file size, iterating over all chunks via shredChunk
// must leave the file size unchanged.
func TestShredChunk_PreservesFileSizeForVariousSizes(t *testing.T) {
	sizes := []int64{
		int64(chunkSize),             // exactly one chunk
		int64(3 * chunkSize),         // multiple exact chunks
		int64(chunkSize) + 512,       // one full + one partial chunk
		int64(7*chunkSize) + 13,      // odd remainder across many chunks
	}

	for _, size := range sizes {
		size := size
		t.Run(fmt.Sprintf("%d_bytes", size), func(t *testing.T) {
			path := createTempFile(t, size)
			defer os.Remove(path)

			f, err := os.OpenFile(path, os.O_RDWR, 0644)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()

			content := bytes.Repeat([]byte{0xAA}, chunkSize)
			numChunks := (size + int64(chunkSize) - 1) / int64(chunkSize)
			for i := int64(0); i < numChunks; i++ {
				if err := shredChunk(f, i*int64(chunkSize), size, content); err != nil {
					t.Fatalf("chunk %d: shredChunk error: %v", i, err)
				}
			}

			stat, err := f.Stat()
			if err != nil {
				t.Fatal(err)
			}
			if stat.Size() != size {
				t.Errorf("size changed: got %d, want %d", stat.Size(), size)
			}
		})
	}
}

// initBits should work for a single chunk only.
func TestBitField_SingleChunk(t *testing.T) {
	var bf bitField
	bf.initBits(int64(chunkSize))

	chunk, err := bf.firstFree()
	if err != nil {
		t.Fatalf("firstFree: %v", err)
	}
	if chunk != 0 {
		t.Errorf("expected chunk 0, got %d", chunk)
	}

	_, err = bf.firstFree()
	if err == nil {
		t.Error("expected error after last chunk was claimed")
	}
}

// initBits should be triggered only once.
func TestBitField_AllChunksReturnedExactlyOnce(t *testing.T) {
	const n = 5
	var bf bitField
	bf.initBits(int64(n * chunkSize))

	seen := make(map[uint64]bool)
	for i := 0; i < n; i++ {
		chunk, err := bf.firstFree()
		if err != nil {
			t.Fatalf("call %d: firstFree: %v", i, err)
		}
		if seen[chunk] {
			t.Errorf("chunk %d returned twice", chunk)
		}
		seen[chunk] = true
	}

	if _, err := bf.firstFree(); err == nil {
		t.Error("expected error after all chunks claimed")
	}
}

// the bitField is an array of 64 bits words. Make sure there are enough words
func TestBitField_PartialLastChunk(t *testing.T) {
	// 1.5 chunks → only 2 usable chunks should be exposed.
	var bf bitField
	bf.initBits(int64(chunkSize) + int64(chunkSize)/2)

	count := 0
	for {
		if _, err := bf.firstFree(); err != nil {
			break
		}
		count++
	}
	if count != 2 {
		t.Errorf("expected 2 usable chunks, got %d", count)
	}
}

// the bitField is an array of 64 bits words. Make sure it adds 1s for unused bits
func TestBitField_PartialLastChunkSetBits(t *testing.T) {
	// 1.5 chunks → only 2 usable chunks should be exposed.
	var bf bitField
	bf.initBits(int64(chunkSize) + int64(chunkSize)/2)

	ones := ^uint64(0)
	mask := ones << 2

	if mask|bf[0] != mask {
		t.Errorf("expected bits 2-63 to be set in last word, got\n0b%b", bf[0])
	}
}

// test workers creation
func TestNewMinionsPool_SetsMinionCount(t *testing.T) {
	for _, n := range []int{1, 2, 8, 16} {
		n := n
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			p := newMinionsPool(n)
			if p == nil {
				t.Fatal("newMinionsPool returned nil")
			}
			if p.minions != n {
				t.Errorf("got p.minions=%d, want %d", p.minions, n)
			}
		})
	}
}

// test the file is truly deleted
func TestShred_DeletesFile(t *testing.T) {
	path := createTempFile(t, int64(2*chunkSize))

	if err := Shred(path); err != nil {
		os.Remove(path)
		t.Fatalf("Shred error: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		os.Remove(path) // cleanup
		t.Errorf("expected %s to be deleted after Shred, but it still exists", path)
	}
}

// test the file is truly deleted
func TestShredPool_DeletesFile(t *testing.T) {
	path := createTempFile(t, int64(2*chunkSize))

	if err := ShredPool(path); err != nil {
		os.Remove(path)
		t.Fatalf("ShredPool error: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		os.Remove(path)
		t.Errorf("expected %s to be deleted after ShredPool, but it still exists", path)
	}
}

// test wrong path given as input
func TestShred_NonexistentFile(t *testing.T) {
	err := Shred("/tmp/__nonexistent_shred_xyz123__")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

// test empty file given as input
func TestShred_EmptyFile(t *testing.T) {
	f, err := os.CreateTemp("", "shred-empty-*")
	if err != nil {
		t.Fatal(err)
	}
	path := f.Name()
	f.Close()
	defer os.Remove(path) // cleanup if Shred skips deletion for empty files

	// Must complete without deadlock or panic.
	if err := Shred(path); err != nil {
		t.Errorf("Shred on empty file returned error: %v", err)
	}
}
