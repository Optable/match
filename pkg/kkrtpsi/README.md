# kkrtpsi implementation

# protocol
The KKRT PSI (Batched-OPRF PSI) [1] is one of the most efficient OT-extension ([oblivious transfer](https://en.wikipedia.org/wiki/Oblivious_transfer)) based PSI protocol that boast more than 100 times speed up in single core performance (350s for DHPSI vs 2.5s for KKRT for a match between two 1 Million records dataset). It is secure against semi-honest adversaries, a malicious party that adheres to the protocol honestly but wants to learn/extract the other party's private information from the data being exchanged.
Discalimer: This is a work in progress.

1. the receiver generates [CuckooHash](https://en.wikipedia.org/wiki/Cuckoo_hashing) parameters and exchange with the sender.
2. the receiver Cuckhash his input sets _Y_.
3. the sender and the receiver agree on base OT (Naor-Pinkas [2] or Simplest [3]) parameters as well as the OT-extension (IKNP [4]) parameters.
4. the receiver acts as the sender in the OT-extension protocol with two matrices _T_ and  _U_ as input, and receives nothing. The matrix _T_ is the encoding of _Y_ with the CuckooHash table/stash and a linear correcitng code _C_, and matrix _U_ is sampled uniformly random.
5. the sender acts as the receiver in the OT-extension protocol with input secret choice bits _s_, and receives matrix _Q_, with columns of _Q_ correspond to either matrix _T_ or _U_ depending on the value of _s_.
6. the receiver evalutes the [OPRF](https://en.wikipedia.org/wiki/Pseudorandom_function_family#Oblivious_pseudorandom_functions) output of his input set _Y_.
7. the sender computes the key _K_ used for the OPRF evaluation of the receiver, and use the same key _K_ to evaluate the OPRF output of his input set _X_.
8. the sender permutes the OPRF result of the key _K_, and input _X_, and sends it to the receiver.
9. the receiver receives the permuted OPRF evaluation of _X_, and compares with his own OPRF evaluation of _Y_, and outputs the intersection.


## data flow
```
             Sender                                                                  Receiver
             X                                                                       Y



Stage 1      Cuckoo Hash        ◄─────────────────CukooHashSetup────────────────     CH(Y)         Stage 1



Stage 2.1    OT + OTExtension   ◄───────────────────OTSender─────────────────────                  Stage 2.1


                                ───────────────────OTReceiver───────────────────►



             K                  ◄──────────────OTExtensionReceiver───────────────


                                ────────────────OTExtensionSender───────────────►    OPRF(K, Y)



Stage 3      H', S'             ────────────────────PsiStage────────────────────►                  Stage 3


CH(Y):      Cuckoo Hash input Y
K:          OPRF keys
OPRF(K, Y): OPRF evaluation of input Y with key K
H':         OPRF evaluation of input X that are put in cuckoo hash table
S':         OPRF evaluation of input X that are put in cuckoo stash
```

## References

[1] V. Kolesnikov, R. Kumaresan, M. Rosulek, N.Trieu. "Efficient Batched Oblivious PRF with Applications to Private Set Intersection." In Proceedings of the 2016 ACM SIGSAC Conference on Computer and Communications Security (pp. 818-829),2016. Paper available here: https://dl.acm.org/doi/pdf/10.1145/2976749.2978381.

[2] M. Naor, B. Pinkas. "Efficient oblivious transfer protocols." In SODA (Vol. 1, pp. 448-457), 2001. Paper available here: https://link.springer.com/content/pdf/10.1007/978-3-662-46800-5_26.pdf

[3] T. Chou, O. Claudio. "The simplest protocol for oblivious transfer." In International Conference on Cryptology and Information Security in Latin America (pp. 40-58). Springer, Cham, 2015. Paper available here: https://eprint.iacr.org/2015/267.pdf

[4] Y. Ishai and J. Kilian and K. Nissim and E. Petrank, Extending Oblivious Transfers Efficiently. https://www.iacr.org/archive/crypto2003/27290145/27290145.pdf
