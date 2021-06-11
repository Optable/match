# kkrtpsi implementation

# protocol
The KKRT PSI (Batched-OPRF PSI) [1] is one of the most efficient OT-extension ([oblivious transfer](https://en.wikipedia.org/wiki/Oblivious_transfer)) based PSI protocol that boast more than 100 times speed up in single core performance (350s for DHPSI vs 2.5s for KKRT for a match between two 1 Million records dataset). It is secure against semi-honest adversaries, a malicious party that adheres to the protocol honestly but wants to learn/extract the other party's private information from the data being exchanged.
Discalimer: This is a work in progress.

1. the receiver generates [CuckooHash](https://en.wikipedia.org/wiki/Cuckoo_hashing) parameters and exchange with the sender.
2. the receiver Cuckhash his input sets _Y_.
3. the sender and the receiver agree on base OT (Naor-Pinkas [2]) parameters as well as the OT-extension (IKNP [3]) parameters.
4. the receiver acts as the sender in the OT-extension protocol with two matrices _T_ and  _U_ as input, and receives nothing. The matrix _T_ is the encoding of _Y_ with the CuckooHash table/stash and a linear correcitng code _C_, and matrix _U_ is sampled uniformly random.
5. the sender acts as the receiver in the OT-extension protocol with input secret choice bits _s_, and receives matrix _Q_, with columns of _Q_ correspond to either matrix _T_ or _U_ depending on the value of _s_.
6. the receiver evalutes the [OPRF](https://en.wikipedia.org/wiki/Pseudorandom_function_family#Oblivious_pseudorandom_functions) output of his input set _Y_.
7. the sender computes the key _K_ used for the OPRF evaluation of the receiver, and use the same key _K_ to evaluate the OPRF output of his input set _X_.
8. the sender permutes the OPRF result of the key _K_, and input _X_, and sends it to the receiver.
9. the receiver receives the permuted OPRF evaluation of _X_, and compares with his own OPRF evaluation of _Y_, and outputs the intersection.


## data flow


## Progress
- [x] KKRT PSI protobuf definition
- [x] KKRT PSI data flow chart
- [x] Cuckoohash pkg implementation
- [ ] Base OT network protocol implementation
- [ ] Naor-Pinkas base OT implementation using OT network implementation
- [ ] IKNP OT extension (KKRT OT extension) ---> to be explored further
- [ ] Glue every components together to implement KKRT PSI

## References

[1] V. Kolesnikov, R. Kumaresan, M. Rosulek, N.Trieu. Efficient Batched Oblivious PRF with Applications to Private Set Intersection. Full version available at https://eprint.iacr.org/2016/799.pdf, and ACM version at https://dl.acm.org/doi/pdf/10.1145/2976749.2978381.

[2] M. Naor, B. Pinkas, Efficient Oblivious Transfer Protocols. https://dl.acm.org/doi/abs/10.5555/365411.365502

[3] Y. Ishai and J. Kilian and K. Nissim and E. Petrank, Extending Oblivious Transfers Efficiently. https://www.iacr.org/archive/crypto2003/27290145/27290145.pdf
