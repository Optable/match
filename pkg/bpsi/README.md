# BPSI implementation

## protocol

The bloomfilter private set intersection (BPSI) is another naive and insecure protocol, but it is highly efficient and has lower communication cost than [NPSI](../npsi/README.md). It is based on [bloomfilter](https://en.wikipedia.org/wiki/Bloom_filter) [1], a probablistic data structure that uses _k_ independent hash functions to compactly represent a set of _n_ elements with only _m_ bits. It supports _O(1)_ set insertion and provides _O(1)_set membership queries at the cost of a small and tunable false positive rate. This means that we can know for certain an element is not in the bloomfilter, and we know an element is in the bloomfilter except with a small false positive probability. 

In the protocol, the sender _P1_ inserts all its elements _X_ into a bloomfilter, and sends it to the receiver _P2_. To compute the intersection, _P2_ needs to simply check the set membership of each of his elements _Y_ with the received bloomfilter.


## data flow

```
Sender (P1)                                       Receiver (P2)
X                                                 Y

BF(X)         ------------------------------>     intersect(Y, BF(X))

BF(X): Bloomfilter bit set of inputs X
```

# References

[1]  Bloom, Burton H. "Space/time trade-offs in hash coding with allowable errors." Communications of the ACM 13.7 (1970): 422-426.