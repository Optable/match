# kkrtpsi implementation

# protocol
The KKRT PSI (Batched-OPRF PSI) [1] is one of the most efficient OT-extension ([oblivious transfer](https://en.wikipedia.org/wiki/Oblivious_transfer)) based PSI protocol that boasts more than 100 times speed up in single core performance (300s for DHPSI vs 3s for KKRT for a match between two 1 million records dataset). It is secure against semi-honest adversaries, a malicious party that adheres to the protocol honestly but wants to learn/extract the other party's private information from the data being exchanged.

1. the sender generates [CuckooHash](https://en.wikipedia.org/wiki/Cuckoo_hashing) parameters and exchange with the receiver.
2. the receiver inserts his input set _Y_ to the Cuckoo Hash Table.
3. the receiver acts as the sender in the OPRF protocol and samples two matrices _T_ and _U_ such that the matrix _T_ is a uniformly random bit matrix, and the matrix _U_ is the Pseudorandom Code (linear correcting code) _C_ on cuckoohashed inputs. The receiver outputs the matrix _T_ as the OPRF evaluation of his inputs _Y_.
4. the sender acts as the receiver in the OPRF protocol with input secret choice bits _s_, and receives matrix _Q_, with columns of _Q_ correspond to either matrix _T_ or _U_ depending on the value of _s_, and outputs the matrix _Q_. Each row of the column _Q_ along with the secret choice bit _s_ serves as the OPRF keys to encode his own input _X_.
5. the sender uses the key _k_ to encode his own input _X_, and sends it to the receiver.
6. the receiver receives the OPRF evaluation of _X_, and compares with his own OPRF evaluation of _Y_, and outputs the intersection.


## data flow
```
             Sender                                                                  Receiver
             X                                                                       Y



Stage 1      Cuckoo Hash                                                              Stage 1
                                ───────────────────CukooHashParam───────────────►     cuckoo.Insert(Y)



Stage 2.1                                                                             Stage 2.1
             oprf.Receive()     ◄─────────────────────T, U───────────────────────     oprf.Send()


             K = Q              ────────────────────────────────────────────────►     OPRF(K, Y) = T



Stage 3      OPRF(K, X)         ────────────────────OPRF(K, X)──────────────────►     Stage 3


K:          OPRF keys
OPRF(K, Y): OPRF evaluation of input Y with key K
```

## References

[1] V. Kolesnikov, R. Kumaresan, M. Rosulek, N.Trieu. "Efficient Batched Oblivious PRF with Applications to Private Set Intersection." In Proceedings of the 2016 ACM SIGSAC Conference on Computer and Communications Security (pp. 818-829),2016. Paper available here: https://dl.acm.org/doi/pdf/10.1145/2976749.2978381.

[2] M. Naor, B. Pinkas. "Efficient oblivious transfer protocols." In SODA (Vol. 1, pp. 448-457), 2001. Paper available here: https://link.springer.com/content/pdf/10.1007/978-3-662-46800-5_26.pdf

[3] T. Chou, O. Claudio. "The simplest protocol for oblivious transfer." In International Conference on Cryptology and Information Security in Latin America (pp. 40-58). Springer, Cham, 2015. Paper available here: https://eprint.iacr.org/2015/267.pdf

[4] Y. Ishai and J. Kilian and K. Nissim and E. Petrank, Extending Oblivious Transfers Efficiently. https://www.iacr.org/archive/crypto2003/27290145/27290145.pdf
