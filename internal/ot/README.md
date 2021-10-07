# Oblivious Transfer (OT)
Oblivious transfer is a cryptographic primitive crucial to building secure multiparty computation protocols. A secure OT protocol allows for two untrusted parties, a sender and a receiver, to perform data exchange in the following way. A sender has as input two messages _M<sub>0</sub>_, _M<sub>1</sub>_, and a receiver has a selection bit _b_. After the OT protocol, the receiver will learn only the message _M<sub>b</sub>_ and not _M<sub>1-b</sub>_, while the sender does not learn the selection bit _b_. This way the receiver does not learn the unintended message (protect against malicious receiver), and the sender cannot forge messages since he does not know which message will be learnt by the receiver (protect against malicious sender).
After 40 years since its invention, two notable base OT protocols are the Naor-Pinkas OT[1] and the Simplest Protocol for OT[2]. An OT extension uses a base OT to exchange short messages in order to achieve many more effective OTs. One of the most efficient OT extensions is the IKNP OT extension [3]. Recently, [4] improved the IKNP OT extension protocol by examining the protocol in a code theoretic lens and extended IKNP into the KKRT, 1 out of n, OT extension.

# protocols
This package implements the following OT, and OT-extensions.

* Base OT
    * Naor-Pinkas[1] using both `crypto/elliptic` and ristretto points on Curve25519
    * Simplest[2] using both `crypto/elliptic` and ristretto points on Curve25519
* OT-extension (1 out of 2)
    * IKNP[3]
    * ALSZ[5]
* OT-extension (1 out of N)
    * KKRT [4]
    * Improved KKRT (Apply ALSZ to KKRT)

## References

[1] M. Naor, B. Pinkas. "Efficient oblivious transfer protocols." In SODA (Vol. 1, pp. 448-457), 2001. Paper available here: https://link.springer.com/content/pdf/10.1007/978-3-662-46800-5_26.pdf

[2] T. Chou, O. Claudio. "The simplest protocol for oblivious transfer." In International Conference on Cryptology and Information Security in Latin America (pp. 40-58). Springer, Cham, 2015. Paper available here: https://eprint.iacr.org/2015/267.pdf

[3] Y. Ishai, J. Kilian, K. Nissim, E. Petrank. "Extending oblivious transfers efficiently." In Annual International Cryptology Conference (pp. 145-161). Springer, Berlin, Heidelberg, 2003. Paper available here: https://www.iacr.org/archive/crypto2003/27290145/27290145.pdf

[4] V. Kolesnikov, R. Kumaresan, M. Rosulek, N.Trieu. "Efficient Batched Oblivious PRF with Applications to Private Set Intersection." In Proceedings of the 2016 ACM SIGSAC Conference on Computer and Communications Security (pp. 818-829),2016. Paper available here: https://dl.acm.org/doi/pdf/10.1145/2976749.2978381

[5] G. Asharov, Y. Lindell, T. Schneider, M. Zohner. "More Efficient Oblivious Transfer Extensions". Source: https://dl.acm.org/doi/10.1007/s00145-016-9236-6
