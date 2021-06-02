# npsi implementation

The naive hashing protocol (npsi) [1] is the most commonly used, but it is an *insecure* solution for PSI. The protocol has a major security flaw if the elements are taken from a domain which does not have very large entropy. In that case, _P2_ (the receiver) can recover all elements in the set of _P1_ (the sender) by running a brute force attack.

In the protocol, _P2_ samples a random 32 bytes salt _K_ and sends it to _P1_. Both parties then use a non cryptographic Hash function ([SIP](https://en.wikipedia.org/wiki/SipHash), [murmur3](https://en.wikipedia.org/wiki/MurmurHash)) to hash their elements salted with _K_. _P1_ then randomly permutes its hash values _H<sub>x</sub>_ and sends them to _P2_, which computes the intersection.

# data flow

```
Sender (P1)                                       Receiver (P2)
X                                                 Y

receive K        <------------------------------  generate K (32 bytes)

sip(K,X) -> H<sub>X</sub>  ------------------------------>  intersect(H<sub>X</sub>, sip(K,Y) -> H<sub>Y</sub>)

sip(K,I): SipHash of input I salted with K
```

# References

[1]  B. Pinkas, T. Schneider, G. Segev, M. Zohner. Phasing: Private Set Intersection using Permutation-based Hashing. USENIX Security 2015. Full version available at http://eprint.iacr.org/2015/634.
