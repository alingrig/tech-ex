package shredder

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"math/bits"
	"errors"
	"crypto/rand"
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
const minPoolChunks = 100


/*************************Random Generation Functions***************************************/
func genRandomContent(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

/*****************************Shredding Functions*******************************************/
func shredChunk(f *os.File, start int64, fileSize int64, content []byte) error {
	if remaining := fileSize - start; int64(len(content)) > remaining {
		content = content[:remaining]
	}

	_, err := f.WriteAt(content, start)
	if err != nil {
		fmt.Println("Write error:", err)
		return err
	}

	return nil
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
		if atomic.CompareAndSwapUint64(&(*bf)[wordIdx], current, current|bitMask) {
			return globalPos, nil
		}
	}
	return 0, errors.New("No free bits left")
}

func (bf *bitField) initBits(fileSize int64) {

	totalChunks := (fileSize + chunkSize - 1) / chunkSize
	chunkBitWords := (totalChunks + bitFieldArrayElemSize - 1) / bitFieldArrayElemSize

    *bf = make(bitField, chunkBitWords)

	if totalChunks == 0 {
        return
    }

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

func (mp *minionsPool) Close(f *os.File) error {
	if mp.waitG != nil {
		mp.waitG.Wait()
	}

	if err := f.Sync(); err != nil {
		fmt.Println("Sync error:", err)
		return err
	}

	return nil
}
