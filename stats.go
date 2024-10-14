package main

type stats struct {
	Name           string `json:"name"`
	MaxDepth       int64  `json:"max_depth"`
	Buckets        int64  `json:"buckets"`
	Keys           int64  `json:"keys"`
	TotalKeySize   int64  `json:"total_key_size"`
	TotalValueSize int64  `json:"total_value_size"`
}
