# shredder

A concurrent file shredder written in Go. Overwrites every byte of a file multiple times with cryptographically random data before deleting it, making recovery significantly harder than a simple `os.Remove`.

---

## How it works

Two shredding strategies are provided:

### `Shred` — Direct mode
Every worker goroutine generates its own random content via `crypto/rand` and writes directly to its claimed chunk. Simple and effective for most use cases.

```
File: [chunk0][chunk1][chunk2][chunk3]...
        ↑       ↑       ↑       ↑
     worker1 worker2 worker3 worker4  (one per CPU core)
```

### `ShredPool` — Producer/Consumer Pool mode
A set of **producer** goroutines pre-generates large random buffers (100 MB each) into a shared channel. A separate set of **consumer** goroutines reads from that channel and writes chunks to disk. Designed to keep the random-number generation and disk I/O pipelines saturated in parallel.

```
Producers (N/2 cores)          Consumers (N/2 cores)
  crypto/rand → 100MB buf  →  [randomChunks channel]  →  WriteAt to file
```

Both modes:
- Split the file into **1 MB chunks**
- Overwrite each chunk **3 times** with random data
- Use a **lock-free bitfield** (CAS on `uint64` words) so workers never write the same chunk twice
- **Delete the file** with `os.Remove` after all overwrites complete successfully

---

## Installation

```bash
go get github.com/alingrig/tech-ex/shredder-task2/shredder
```

---

## Usage

```go
import "github.com/alingrig/tech-ex/shredder-task2/shredder"

// Direct mode — one goroutine per CPU core
if err := shredder.Shred("/path/to/secret.txt"); err != nil {
    log.Fatal(err)
}

// Pool mode — producers pre-generate random data, consumers write
if err := shredder.ShredPool("/path/to/secret.txt"); err != nil {
    log.Fatal(err)
}
```

Both functions return `nil` on success. The file is deleted automatically. On error the file is left in place (partially overwritten).

---

## Configuration

Constants in `shredder-common.go`:

| Constant | Default | Description |
|---|---|---|
| `chunkSize` | `1 MB` | Size of each write unit |
| `overwrites` | `3` | Number of random overwrite passes per chunk |
| `randomContentMB` | `100` | Size of each producer buffer (pool mode) |
| `minPoolChunks` | `100` | Minimum chunks buffered before triggering refill |

---

## Architecture

### Bitfield

The file is divided into N chunks. A `[]uint64` bitfield tracks which chunks have been claimed. Workers race to claim chunks using **atomic Compare-And-Swap (CAS)** — no mutex needed for chunk assignment.

```
Word 0:  [bit63 ... bit1 bit0]   ← 0 = free, 1 = claimed
Word 1:  [bit63 ... bit1 bit0]
...
```

Bits beyond the last valid chunk are pre-set to `1` (used) during `initBits` so workers never write past end-of-file.

---

## Running tests

```bash
# All tests
go test ./...

# With race detector (recommended)
go test -race ./...

# Verbose output
go test -v ./...

# Specific test
go test -run TestShred_DeletesFile -v ./...
```

---
