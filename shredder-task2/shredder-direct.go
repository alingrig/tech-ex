//go:build direct

package main

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"math/bits"
	"errors"
	"crypto/rand"
	"runtime"
)

/************************Global Declarations and Definitions********************************/
type bitField []uint64

type minionsPool struct {
	minions 	int
	bitField 	bitField
	lock		sync.Mutex
	waitG		*sync.WaitGroup
}

const chunkSize = 1024 * 1024 // 1MB chunks - good performance on most mediums
const overwrites = 3
const bitFieldArrayElemSize = 64
const workersNo = 4
const randomContentMB = 100
const randomContentSize = randomContentMB * 1024 * 1024 // 100MB

var randomChunks  = make(chan []byte, randomContentMB)
var triggerRefill = make(chan struct{}, 1)

var done atomic.Uint64

/*************************Random Generation Functions***************************************/
func genRandomContent(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, err
}

/*****************************Shredding Functions*******************************************/
func shredChunk(f *os.File, start int64, fileSize int64, content []byte) {
	if remaining := fileSize - start; int64(len(content)) > remaining {
		content = content[:remaining]
	}

	_, err := f.WriteAt(content, start)
	if err != nil {
		fmt.Println("Write error:", err)
		return
	}
}

/**********************************BitField functions***************************************/
func (bf *bitField) firstFree() (uint64, error) {
	for wordIdx := 0; wordIdx < len(*bf); wordIdx++ {
		current := atomic.LoadUint64(&(*bf)[wordIdx])
		if current == ^uint64(0) {  // Full word, skip
			continue
		}

		// Find first free bit in this word
		bitIdx := bits.TrailingZeros64(^current)
		globalPos := uint64(wordIdx) * bitFieldArrayElemSize + uint64(bitIdx)
		if globalPos >= uint64(len(*bf)*bitFieldArrayElemSize) {
			continue
		}

		// Try to claim it with CAS
		bitMask := uint64(1) << bitIdx
		expected := current
		if atomic.CompareAndSwapUint64(&(*bf)[wordIdx], expected, expected|bitMask) {
			return globalPos, nil
		}
	}
	return 0, errors.New("No free bits left")
}

func (bf *bitField) initBits(fileSize int64) {

	totalChunks := (fileSize + chunkSize - 1) / chunkSize
	chunkBitWords := (totalChunks + bitFieldArrayElemSize - 1) / bitFieldArrayElemSize

    *bf = make(bitField, chunkBitWords)

	// Mark all bits beyond totalBits - 1 as used (1)
	lastWordIdx := (totalChunks - 1) / bitFieldArrayElemSize
	firstUnusedBitInLastWord := (totalChunks - 1) % bitFieldArrayElemSize + 1
	if firstUnusedBitInLastWord < bitFieldArrayElemSize {
		mask := ^uint64(0) << firstUnusedBitInLastWord
		(*bf)[lastWordIdx] |= mask
	}
}

/*****************************Minions Pool functions****************************************/
func newMinionsPool(minions int) *minionsPool {
	return &minionsPool{
		minions: minions,
	}
}

func executeTask(minionID int, f *os.File, fileSize int64, waitG *sync.WaitGroup, mp *minionsPool) {
	defer waitG.Done()
	for chunk, err := mp.bitField.firstFree(); err == nil; chunk, err = mp.bitField.firstFree() {
		//fmt.Printf("Minion %d shredding chunk %d\n", minionID, chunk)

		for pass := 0; pass < overwrites; pass++ {
			content, _ := genRandomContent(chunkSize)
			shredChunk(f, int64(chunk * chunkSize), fileSize, content)
		}
	}
}

func (mp *minionsPool) Start(f *os.File) {
	stat, err := f.Stat()
	if err != nil {
		fmt.Println("Stat error:", err)
		return
	}

	if(stat.Size() == 0) {
		fmt.Println("Empty file")
		return
	}

	mp.bitField.initBits(stat.Size())

	fileSize := stat.Size()
	mp.waitG = &sync.WaitGroup{}
	for minionID := 1; minionID <= mp.minions; minionID++ {
		mp.waitG.Add(1)
		go executeTask(minionID, f, fileSize, mp.waitG, mp)
	}
}

func (mp *minionsPool) Close(f *os.File) {
	if mp.waitG != nil {
		mp.waitG.Wait()
	}
	if err := f.Sync(); err != nil {
		fmt.Println("Sync error:", err)
	}
}

/**************************************Main function****************************************/
func main() {
	cores := runtime.NumCPU()
	pool := newMinionsPool(cores)

	fmt.Printf("CPU cores: %d\n", cores)

	name := "test.bin"
	f, err := os.OpenFile(name, os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Open error:", err)
		return
	}
	defer f.Close()

	pool.Start(f)

	pool.Close(f)
	fmt.Println("All shreds done!")
}
