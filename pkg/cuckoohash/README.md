# Cuckoo hash tables

# protocol
Cuckoo hash tables [1] is an optimized hash table data structure with O(1) look up times in worst case scenario, and O(1) insertion time with amortized costs. We implement a variant of cuckoo hash tables that uses *3* hash functions and a stash to limit the probability of hashing faillure to _2<sup>-\sigma</sup>_, where _\sigma_ is a security parameter that is set to _40_.
Discalimer: This is a work in progress.

## References

[1] Pagh, R., and Rodler, F. F. Cuckoo hashing. J. Algorithms 51, 2 (2004), 122–144.
