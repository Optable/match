# npsi implementation

The naive hashing protocol (npsi) [1] is the most commonly used, but it is an *insecure* solution for PSI. The protocol has a major security flaw if the elements are taken from a domain which does not have very large entropy. In that case, _P<sub>2</sub>_ (the receiver) can recover all elements in the set of _P<sub>1</sub>_ (the sender) by running a brute force attack.

In the protocol, _P<sub>2</sub>_ samples a random 32 bytes salt _K_ and sends it to _P<sub>1</sub>_. Both parties then use a non cryptographic Hash function ([SIP](https://en.wikipedia.org/wiki/SipHash), [murmur3](https://en.wikipedia.org/wiki/MurmurHash)) to hash their input identifiers salted with _K_. _P<sub>1</sub>_ then randomly permutes its hash values _H<sub>x</sub>_ and sends them to _P<sub>2</sub>_, who computes the intersection of both hashed identifiers.

# data flow

```
Sender (P1)                                       Receiver (P2)
X                                                 Y

receive K        <------------------------------  generate K (32 bytes)

sip(K,X) -> H_X  ------------------------------>  intersect(H_X, sip(K,Y) -> H_Y)

sip(K,I): SipHash of input I salted with K
```

# References

[1]  B. Pinkas, T. Schneider, G. Segev, M. Zohner. Phasing: Private Set Intersection using Permutation-based Hashing. USENIX Security 2015. Full version available at http://eprint.iacr.org/2015/634.
