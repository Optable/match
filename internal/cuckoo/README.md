# Cuckoo hash tables

## Description
Cuckoo hash tables [1] is an optimized hash table data structure with O(1) look up times in worst case scenario, and O(1) insertion time with amortized costs. We implement a variant of cuckoo hash tables that uses *3* hash functions and a stash to limit the probability of hashing faillure to _2<sup>σ</sup>_, where _σ_ is a security parameter that is set to _40_.

## Benchmark
```
go test -bench=. --benchmem -v ./internal/cuckoo/ --count=1
=== RUN   TestStashSize
--- PASS: TestStashSize (0.00s)
=== RUN   TestNewCuckoo
--- PASS: TestNewCuckoo (0.00s)
=== RUN   TestInsertAndGetHashIdx
    cuckoo_test.go:88: To be inserted: 1000000, bucketSize: 2000000, load factor: 0.500000, failure insertion:  0, stashSize: 3, items on stash: 0
--- PASS: TestInsertAndGetHashIdx (1.69s)
goos: darwin
goarch: amd64
pkg: github.com/optable/match/internal/cuckoo
cpu: Intel(R) Core(TM) i7-8850H CPU @ 2.60GHz
BenchmarkCuckooInsert
BenchmarkCuckooInsert-12        	 1805648	       679.2 ns/op	       0 B/op	       0 allocs/op
BenchmarkCuckooGetHashIdx
BenchmarkCuckooGetHashIdx-12    	 3776962	       422.4 ns/op	       0 B/op	       0 allocs/op
BenchmarkCuckooExists
BenchmarkCuckooExists-12        	 3719437	       424.2 ns/op	       0 B/op	       0 allocs/op
PASS
ok  	github.com/optable/match/internal/cuckoo	16.396s
```

## References

[1] Pagh, R., and Rodler, F. F. Cuckoo hashing. J. Algorithms 51, 2 (2004), 122–144.
