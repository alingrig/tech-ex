package shredder

import (
	"fmt"
	"os"
	"sync"
	"runtime"
)

/*************************Random Generation Functions***************************************/
func startProducers(n int, stopProducers chan struct{}, errCh chan<- error) {
	randomChunks = make(chan []byte, n * randomContentMB) // n×100 buffer
	for i := 0; i < n; i++ {
		go func() {
			for {
				select {
					case <- stopProducers:
						errCh <- nil
						return
					default:
				}
				buf, err := genRandomContent(randomContentSize)
				if err != nil {
					errCh <- fmt.Errorf("producer: genRandomContent: %w", err)
					return
				}
				for j := 0; j < randomContentMB; j++ {
					randomChunks <- buf[j*chunkSize : (j+1)*chunkSize]
				}
			}
		}()
	}
}

func getRandomChunk() []byte {
    chunk := <-randomChunks
    if len(randomChunks) <= minPoolChunks {
        select {
        case triggerRefill <- struct{}{}:
        default:
        }
    }
    return chunk
}

/*****************************Minions Pool functions****************************************/
func executePoolTask(minionID int, f *os.File, fileSize int64, waitG *sync.WaitGroup, mp *minionsPool, errCh chan<- error) {
	defer waitG.Done()
	for chunk, err := mp.bitField.firstFree(); err == nil; chunk, err = mp.bitField.firstFree() {
		//fmt.Printf("Minion %d shredding chunk %d\n", minionID, chunk)

		for pass := 0; pass < overwrites; pass++ {
			content := getRandomChunk()
			err := shredChunk(f, int64(chunk * chunkSize), fileSize, content)
			if err != nil {
				errCh <- fmt.Errorf("minion %d, pass %d, sharedChunk: %w", minionID, pass, err)
				return
			}
		}
	}

	errCh <- nil
}

func (mp *minionsPool) StartPool(f *os.File, stopProducers chan struct{}, errCh chan<- error) {
	stat, err := f.Stat()
	if err != nil {
		errCh <- fmt.Errorf("ShredPool: Stat: %w", err)
		return
	}

	if stat.Size() == 0 {
		fmt.Println("Empty file")
		return
	}

	mp.bitField.initBits(stat.Size())

    numProducers := max(1, runtime.NumCPU()/2)
	fmt.Printf("Producers CPU cores: %d\n", numProducers)
    startProducers(numProducers, stopProducers, errCh)

	fileSize := stat.Size()
	mp.waitG = &sync.WaitGroup{}
	for minionID := 1; minionID <= mp.minions; minionID++ {
		mp.waitG.Add(1)
		go executePoolTask(minionID, f, fileSize, mp.waitG, mp, errCh)
	}
}

/**************************************Main function****************************************/
func ShredPool(path string) error {
	stopProducers := make(chan struct{})
	cores := runtime.NumCPU()
	numConsumers := max(1, cores/2)
	pool := newMinionsPool(numConsumers)

	fmt.Printf("Consumers CPU cores: %d\n", numConsumers)

	f, err := os.OpenFile(path, os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("shredder:", err)
		return err
	}
	defer f.Close()

	errCh := make(chan error, cores)

	pool.StartPool(f, stopProducers, errCh)

	defer close(stopProducers)

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
			fmt.Println("Remove shredded file: %w", err)
			return err
		}
	}

	fmt.Println("All shreds done!")

	return firstErr
}
