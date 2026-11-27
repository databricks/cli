package acceptance_test

import (
	"errors"
	"fmt"
	"hash/fnv"
	"os"
	"strconv"
	"strings"
	"testing"
)

// ShardEnvVar ("i/N", 0-based) runs only the subtests hashed into shard i of N, so parallel
// jobs can split the suite. Every subtest lands in exactly one shard.
const ShardEnvVar = "TESTS_SELECT_SHARD"

type shardSelector struct {
	index int
	count int
}

func newShardSelector(t *testing.T) shardSelector {
	raw := os.Getenv(ShardEnvVar)
	if raw == "" {
		return shardSelector{}
	}
	s, err := parseShard(raw)
	if err != nil {
		t.Fatalf("Invalid %s=%q: %v", ShardEnvVar, raw, err)
	}
	return s
}

func parseShard(raw string) (shardSelector, error) {
	is, ns, ok := strings.Cut(raw, "/")
	if !ok {
		return shardSelector{}, errors.New("expected i/N")
	}
	i, err := strconv.Atoi(is)
	if err != nil {
		return shardSelector{}, err
	}
	n, err := strconv.Atoi(ns)
	if err != nil {
		return shardSelector{}, err
	}
	if n < 1 || i < 0 || i >= n {
		return shardSelector{}, errors.New("expected 0 <= i < N")
	}
	return shardSelector{index: i, count: n}, nil
}

// skipReason returns a non-empty skip message if the subtest belongs to another shard.
func (s shardSelector) skipReason(dir string, envset []string) string {
	if s.count == 0 {
		return ""
	}
	h := fnv.New64a()
	h.Write([]byte(subtestID(dir, envset)))
	if int(h.Sum64()%uint64(s.count)) == s.index {
		return ""
	}
	return fmt.Sprintf("Skipped by %s=%d/%d", ShardEnvVar, s.index, s.count)
}
