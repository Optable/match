# npsi implementation

## protocol

In the naive private set intersection (NPSI) [1], both parties agree on a non-cryptographic hash function, apply it to their inputs and then compare the resulting hashes. It is the most commonly used protocol due to its efficiency and ease for implementation, but it is *insecure*. The protocol has a major security flaw if the elements are taken from a small domain or a domain that does not have high entropy. In that case, _P<sub>2</sub>_ (the receiver) can recover all elements in the set of _P<sub>1</sub>_ (the sender) by running a brute force attack.

In the protocol, _P<sub>2</sub>_ samples a random 32 bytes salt _K_ and sends it to _P<sub>1</sub>_. Both parties then use a non-cryptographic hash function ([MetroHash](http://www.jandrewrogers.com/2015/05/27/metrohash/)) to hash their input identifiers seeded with _K_. _P<sub>1</sub>_ sends the hash values _H<sub>x</sub>_ to _P<sub>2</sub>_, who computes the intersection of both hashed identifiers.

## data flow

```
Sender (P1)                                       Receiver (P2)
X                                                 Y

receive K        <------------------------------  generate K (32 bytes)

mh(K,X) -> H_X  ------------------------------>  intersect(H_X, mh(K,Y) -> H_Y))

mh(K,I): Metro hash of input I seeded with K
```

# References

[1]  B. Pinkas, T. Schneider, G. Segev, M. Zohner. Phasing: Private Set Intersection using Permutation-based Hashing. USENIX Security 2015. Full version available at http://eprint.iacr.org/2015/634.
