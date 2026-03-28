package shredder

import (
	"fmt"
	"os"
	"sync"
	"runtime"
)

var stopProducers = make(chan struct{})

/*************************Random Generation Functions***************************************/
func startProducers(n int) {
	randomChunks = make(chan []byte, n * randomContentMB) // n×100 buffer
	for i := 0; i < n; i++ {
		go func() {
			for {
				select {
					case <- stopProducers:
						return
					default:
				}
				buf, err := genRandomContent(randomContentSize)
				if err != nil {
					continue
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
func executePoolTask(minionID int, f *os.File, fileSize int64, waitG *sync.WaitGroup, mp *minionsPool) {
	defer waitG.Done()
	for chunk, err := mp.bitField.firstFree(); err == nil; chunk, err = mp.bitField.firstFree() {
		//fmt.Printf("Minion %d shredding chunk %d\n", minionID, chunk)

		for pass := 0; pass < overwrites; pass++ {
			content := getRandomChunk()
			shredChunk(f, int64(chunk * chunkSize), fileSize, content)
		}
	}
}

func (mp *minionsPool) StartPool(f *os.File) {
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

    numProducers := max(1, runtime.NumCPU()/2)
	fmt.Printf("Producers CPU cores: %d\n", numProducers)
    startProducers(numProducers)

	fileSize := stat.Size()
	mp.waitG = &sync.WaitGroup{}
	for minionID := 1; minionID <= mp.minions; minionID++ {
		mp.waitG.Add(1)
		go executePoolTask(minionID, f, fileSize, mp.waitG, mp)
	}
}

/**************************************Main function****************************************/
func ShredPool(path string) error {
	numConsumers := max(1, runtime.NumCPU()/2)
	pool := newMinionsPool(numConsumers)

	fmt.Printf("Consumers CPU cores: %d\n", numConsumers)

	f, err := os.OpenFile(path, os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("shredder: %w", err)
	}
	defer f.Close()

	pool.StartPool(f)

	pool.Close(f)
	fmt.Println("All shreds done!")

	close(stopProducers)

	return nil
}
