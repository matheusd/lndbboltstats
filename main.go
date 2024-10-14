package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"
	"time"

	"go.etcd.io/bbolt"
)

func log(s string, args ...any) {
	const dateTime = "2006-01-02 15:04:05.000:"
	fmt.Println(time.Now().Format(dateTime), fmt.Sprintf(s, args...))
}

func readBucket(tx *bbolt.Tx, b *bbolt.Bucket, s *stats, depth int64) {
	s.MaxDepth = max(s.MaxDepth, depth)
	c := b.Cursor()
	for k, v := c.First(); k != nil; k, v = c.Next() {
		if v == nil {
			// Possibly inner bucket.
			s.Buckets++
			bb := b.Bucket(k)
			if bb == nil {
				// Key with empty value.
				s.Keys++
				s.TotalKeySize += int64(len(k))
				continue
			}

			readBucket(tx, bb, s, depth+1)
			continue
		}

		s.Keys++
		s.TotalKeySize += int64(len(k))
		s.TotalValueSize += int64(len(v))
	}
}

func realMain() error {
	if len(os.Args) < 2 {
		return errors.New("Usage: lndbboltstats <filename>")
	}

	filename := os.Args[1]
	log("Using DB: %s", filename)

	stat, err := os.Stat(filename)
	if err != nil {
		return err
	}
	dbSize := stat.Size()
	log("Total DB file size: %s", humanizeBytes(dbSize))

	db, err := bbolt.Open(filename, 0600, &bbolt.Options{Timeout: 5 * time.Second, NoFreelistSync: false, ReadOnly: true})
	if err != nil {
		return fmt.Errorf("Unable to open DB: %v", err)
	}
	defer func() {
		err := db.Close()
		if err != nil {
			log("Error closing db: %v", err)
		}
	}()

	tx, err := db.Begin(false)
	if err != nil {
		return err
	}
	defer func() {
		err := tx.Rollback()
		if err != nil {
			log("Error rolling back tx: %v", err)
		}
	}()

	// Look at the top level buckets, count them and record the non-arb-log
	// ones.
	var topLevelBuckets []string
	var topLevelBucketKeys [][]byte
	var nbArbLogBuckets int64
	c := tx.Cursor()
	for k, _ := c.First(); k != nil; k, _ = c.Next() {
		if len(k) == 68 {
			nbArbLogBuckets++
		} else {
			name := quoteKey(slices.Clone(k))
			topLevelBucketKeys = append(topLevelBucketKeys, k)
			topLevelBuckets = append(topLevelBuckets, name)
		}
	}
	sort.Slice(topLevelBuckets, func(i, j int) bool {
		return topLevelBuckets[i] < topLevelBuckets[j]
	})
	log("Found %d top-level buckets and %d arb log buckets", len(topLevelBuckets),
		nbArbLogBuckets)
	log("Top Level Buckets: %s", strings.Join(topLevelBuckets, ", "))

	log("Reading top level buckets...")
	var globalStats stats = stats{Name: "global-stats"}
	for _, k := range topLevelBucketKeys {
		s := stats{Name: quoteKey(slices.Clone(k))}
		b := tx.Bucket(k)
		if b == nil {
			log("%s is not a bucket", s.Name)
			continue
		}
		readBucket(tx, b, &s, 1)

		jsonStats, err := json.Marshal(s)
		if err != nil {
			return err
		}
		log(string(jsonStats))
		globalStats.Buckets += s.Buckets
		globalStats.Keys += s.Keys
		globalStats.MaxDepth = max(globalStats.MaxDepth, s.MaxDepth)
		globalStats.TotalKeySize += s.TotalKeySize
		globalStats.TotalValueSize += s.TotalValueSize
	}

	var arbStats stats = stats{Name: "arbitrator-logs"}
	for k, _ := c.First(); k != nil; k, _ = c.Next() {
		if len(k) != 68 {
			continue
		}

		b := tx.Bucket(k)
		if b == nil {
			continue
		}
		arbStats.Buckets++
		readBucket(tx, b, &arbStats, 1)
	}

	jsonStats, err := json.Marshal(arbStats)
	if err != nil {
		return err
	}
	log(string(jsonStats))

	globalStats.Buckets += arbStats.Buckets
	globalStats.Keys += arbStats.Keys
	globalStats.MaxDepth = max(globalStats.MaxDepth, arbStats.MaxDepth)
	globalStats.TotalKeySize += arbStats.TotalKeySize
	globalStats.TotalValueSize += arbStats.TotalValueSize

	jsonStats, err = json.Marshal(globalStats)
	if err != nil {
		return err
	}
	log(string(jsonStats))

	log("Totals:")
	log("Max Depth: %d", globalStats.MaxDepth)
	log("Buckets: %d", globalStats.Buckets)
	log("Keys: %d", globalStats.Keys)
	log("Total Key Size: %s", humanizeBytes(globalStats.TotalKeySize))
	log("Total Value Size: %s", humanizeBytes(globalStats.TotalValueSize))
	log("Total DB file size: %s", humanizeBytes(dbSize))

	return nil
}

func main() {
	err := realMain()
	if err != nil {
		fmt.Println("Error:", err.Error())
		os.Exit(1)
	}
}
