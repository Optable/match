# Cuckoo hash tables

## Description
Cuckoo hash tables [1] is an optimized hash table data structure with O(1) look up times in worst case scenario, and O(1) insertion time with amortized costs. We implement a variant of cuckoo hash tables that uses *3* hash functions and a stash to limit the probability of hashing faillure to _2<sup>σ</sup>_, where _σ_ is a security parameter that is set to _40_.

## Benchmark
```
go test -v -bench=. --benchmem ./internal/cuckoo/
=== RUN   TestNewCuckoo
--- PASS: TestNewCuckoo (0.00s)
=== RUN   TestInsertAndGetHashIdx
    cuckoo_test.go:61: To be inserted: 1000000, bucketSize: 1400000, load factor: 0.714286, failure insertion:  0, taken 657.621072ms
--- PASS: TestInsertAndGetHashIdx (1.15s)
goos: darwin
goarch: amd64
pkg: github.com/optable/match/internal/cuckoo
cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
BenchmarkCuckooInsert
BenchmarkCuckooInsert-12         1929448               555.0 ns/op             0 B/op          0 allocs/op
BenchmarkCuckooExists
BenchmarkCuckooExists-12         3647469               330.4 ns/op             0 B/op          0 allocs/op
PASS
ok      github.com/optable/match/internal/cuckoo        5.146s
```

## References

[1] Pagh, R., and Rodler, F. F. Cuckoo hashing. J. Algorithms 51, 2 (2004), 122–144.
