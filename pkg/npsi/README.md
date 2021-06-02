# npsi implementation

The naive hashing protocol is the most commonly used, but it is an *insecure* solution for PSI. The protocol has a major security flaw if the elements are taken from a domain which does not have very large entropy. In that case, P2 (the receiver) can recover all elements of in the set of P1 by running a brute force attack.

In the protocol, P2 samples a random 2k-bit salt K and sends it to P1. Both parties then use a non cryptographic Hash function (SIP, murmur3) to hash their elements salted with K. P1 (the sender) then randomly permutes its hash values Hi and sends them to P2, which computes the intersection.

# data flow

```
Sender (p1)                                    Receiver (p2)


receive k        <------------------------------  generate k (32 bytes)

sip(k,x) -> H_x  ------------------------------>  intersect(H_x, sip(k,y) -> H_y)
```