package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"
)

func main() {
	mode := flag.String("type", "sequential", "Trace type: sequential or zipfian")
	events := flag.Int("events", 100000, "Number of events")
	keys := flag.Int("keys", 1000, "Number of unique keys")
	out := flag.String("out", "trace.csv", "Output file")
	flag.Parse()

	f, err := os.Create(*out)
	if err != nil {
		log.Fatalf("create file: %v", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	w.Write([]string{"timestamp_ns", "key"})

	r := rand.New(rand.NewSource(42))
	zipf := rand.NewZipf(r, 1.1, 1.0, uint64(*keys-1))

	var currentKey int
	ts := time.Now().UnixNano()

	for i := 0; i < *events; i++ {
		var keyStr string
		if *mode == "sequential" {
			keyStr = fmt.Sprintf("key_%d", currentKey)
			currentKey++
			if currentKey >= *keys {
				currentKey = 0
			}
		} else {
			keyIdx := zipf.Uint64()
			keyStr = fmt.Sprintf("key_%d", keyIdx)
		}

		w.Write([]string{strconv.FormatInt(ts, 10), keyStr})
		ts += 1000000
	}
}
