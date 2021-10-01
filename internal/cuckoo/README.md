# Cuckoo hash tables

## Description
Cuckoo hash tables [1] is an optimized hash table data structure with O(1) look up times in worst case scenario, and O(1) insertion time with amortized costs. We implement a variant of cuckoo hash tables that uses *3* hash functions and a stash to limit the probability of hashing faillure to _2<sup>σ</sup>_, where _σ_ is a security parameter that is set to _40_.

## Benchmark
```
go test -v -bench=. --benchmem ./internal/cuckoo/
=== RUN   TestNewCuckoo
--- PASS: TestNewCuckoo (0.00s)
=== RUN   TestInsertAndGetHashIdx
    cuckoo_test.go:60: To be inserted: 1000000, bucketSize: 1400000, load factor: 0.714286, failure insertion:  0
--- PASS: TestInsertAndGetHashIdx (0.89s)
goos: darwin
goarch: amd64
pkg: github.com/optable/match/internal/cuckoo
cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
BenchmarkCuckooInsert
BenchmarkCuckooInsert-12        	 2886076	       347.0 ns/op	       0 B/op	       0 allocs/op
BenchmarkCuckooGetHashIdx
BenchmarkCuckooGetHashIdx-12    	 4017060	       303.1 ns/op	       0 B/op	       0 allocs/op
BenchmarkCuckooExists
BenchmarkCuckooExists-12        	 4015934	       305.6 ns/op	       0 B/op	       0 allocs/op
PASS
ok  	github.com/optable/match/internal/cuckoo	6.142s
```

## References

[1] Pagh, R., and Rodler, F. F. Cuckoo hashing. J. Algorithms 51, 2 (2004), 122–144.
