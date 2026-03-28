package shredder

import (
	"fmt"
	"os"
	"sync"
	"runtime"
)

/*****************************Minions Pool functions****************************************/
func executeDirectTask(minionID int, f *os.File, fileSize int64, waitG *sync.WaitGroup, mp *minionsPool) {
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
		go executeDirectTask(minionID, f, fileSize, mp.waitG, mp)
	}
}

/**************************************Main function****************************************/
func Shred(path string) error {
	cores := runtime.NumCPU()
	pool := newMinionsPool(cores)

	fmt.Printf("CPU cores: %d\n", cores)

	f, err := os.OpenFile(path, os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("shredder: %w", err)
	}
	defer f.Close()

	pool.Start(f)

	pool.Close(f)
	fmt.Println("All shreds done!")

	return nil
}
