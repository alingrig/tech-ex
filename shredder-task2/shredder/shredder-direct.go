package shredder

import (
	"fmt"
	"os"
	"sync"
	"runtime"
)

/*****************************Minions Pool functions****************************************/
func executeDirectTask(minionID int, f *os.File, fileSize int64, waitG *sync.WaitGroup, mp *minionsPool, errCh chan<- error) {
	defer waitG.Done()
	for chunk, err := mp.bitField.firstFree(); err == nil; chunk, err = mp.bitField.firstFree() {
		//fmt.Printf("Minion %d shredding chunk %d\n", minionID, chunk)

		for pass := 0; pass < overwrites; pass++ {
			content, err := genRandomContent(chunkSize)
			if err != nil {
				errCh <- fmt.Errorf("minion %d, pass %d, genRandomContent: %w", minionID, pass, err)
				return
			}
			err = shredChunk(f, int64(chunk * chunkSize), fileSize, content)
			if err != nil {
				errCh <- fmt.Errorf("minion %d, pass %d, sharedChunk: %w", minionID, pass, err)
				return
			}
		}
	}

	errCh <- nil
}

func (mp *minionsPool) Start(f *os.File, errCh chan<- error) {
	stat, err := f.Stat()
	if err != nil {
		errCh <- fmt.Errorf("Shred: Stat: %w", err)
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
		go executeDirectTask(minionID, f, fileSize, mp.waitG, mp, errCh)
	}
}

/**************************************Main function****************************************/
func Shred(path string) error {
	cores := runtime.NumCPU()
	pool := newMinionsPool(cores)

	fmt.Printf("CPU cores: %d\n", cores)

	f, err := os.OpenFile(path, os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("shredder:", err)
		return err
	}
	defer f.Close()

	errCh := make(chan error, cores)

	pool.Start(f, errCh)

	err = pool.Close(f)
	if err != nil {
		fmt.Println("Close:", err)
		return err
	}

	close(errCh)

	var firstErr error
	for err := range errCh {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	if firstErr == nil {
		if err := os.Remove(path); err != nil { // delete shredded file
			fmt.Println("Remove shredded file:", err)
			return err
		}
	}

	fmt.Println("All shreds done!")

	return firstErr
}
