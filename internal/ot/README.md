# Oblivious Transfer (OT)
## Introduction
Oblivious transfer is a cryptographic primitive crucial to building secure multiparty computation (MPC) protocols. A secure OT protocol allows for two untrusted parties, a sender and a receiver, to perform data exchange in the following way. A sender has as input two messages _M<sub>0</sub>_, _M<sub>1</sub>_, and a receiver has a selection bit _b_. After the OT protocol, the receiver will learn only the message _M<sub>b</sub>_ and not _M<sub>1-b</sub>_, while the sender does not learn the selection bit _b_. This way the receiver does not learn the unintended message (protect against malicious receiver), and the sender cannot forge messages, since he does not know which message will be learnt by the receiver (protect against malicious sender).
After 40 years since its invention, two notable base OT protocols are the Naor-Pinkas OT[1] and the Simplest Protocol for OT[2].

## Implementation
the following OT protocols are implemented:
* Naor-Pinkas[1] using both `crypto/elliptic` and ristretto points on Curve25519
* Simplest[2] using both `crypto/elliptic` and ristretto points on Curve25519


## References

[1] M. Naor, B. Pinkas. "Efficient oblivious transfer protocols." In SODA (Vol. 1, pp. 448-457), 2001. Paper available here: https://link.springer.com/content/pdf/10.1007/978-3-662-46800-5_26.pdf

[2] T. Chou, O. Claudio. "The simplest protocol for oblivious transfer." In International Conference on Cryptology and Information Security in Latin America (pp. 40-58). Springer, Cham, 2015. Paper available here: https://eprint.iacr.org/2015/267.pdf
