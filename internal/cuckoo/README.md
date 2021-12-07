# Cuckoo hash tables

## Description
Cuckoo hash tables [1] is an optimized hash table data structure with O(1) look up times in worst case scenario, and O(1) insertion time with amortized costs. We implement a variant of cuckoo hash tables that uses *3* hash functions to limit the probability of hashing faillure to _2<sup>σ</sup>_, where _σ_ is a security parameter that is set to _40_.

## Benchmark
```
go test -bench=. -benchmem ./internal/cuckoo/...                                
goos: darwin
goarch: amd64
pkg: github.com/optable/match/internal/cuckoo
cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
BenchmarkCuckooInsert-12         2627492               498.0 ns/op             0 B/op          0 allocs/op
BenchmarkCuckooExists-12         7582648               203.0 ns/op             0 B/op          0 allocs/op
PASS
ok      github.com/optable/match/internal/cuckoo        7.896s
```

## References

[1] Pagh, R., and Rodler, F. F. Cuckoo hashing. J. Algorithms 51, 2 (2004), 122–144.
