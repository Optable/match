# Oblivious Transfer (OT)

# protocol
Oblivious transfer is a cryptographic primitive crucial to building secure multiparty computation protocols. A secure OT protocol allows for two untrusted parties, a sender and a receiver, to perform data exchange in the following way. A sender has as input two messages _M<sub>0</sub>_, _M<sub>1</sub>_, and a receiver has a selection bit _b_. After the OT protocol, the receiver will learn only the message _M<sub>b</sub>_ and not _M<sub>1-b</sub>_, while the sender does not learn the selection bit _b_. This way the receiver does not learn the unintended message (protect against malicious receiver), and the sender cannot forge messages since he does not know which message will be learnt by the receiver (protect against malicious sender).
After 40 years since its invention, two notable base OT protocols are the Naor-Pinkas OT[1] and the Simplest Protocol for OT[2], which are implemented in this package.
Discalimer: This is a work in progress.

## References

[1] Naor, Moni, and Benny Pinkas. "Efficient oblivious transfer protocols." SODA. Vol. 1. 2001.

[2] Chou, Tung, and Claudio Orlandi. "The simplest protocol for oblivious transfer." International Conference on Cryptology and Information Security in Latin America. Springer, Cham, 2015. eprint version: https://eprint.iacr.org/2015/267.pdf
