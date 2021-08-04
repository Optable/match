# Oblivious Pseudorandom Function (OPRF)

# Description
An Oblivious Pseudorandom Function (OPRF) is a two-party protocol for computing the output of a PRF. A pseudorandom function (PRF) <i>F(k, x)</i> is an efficiently computable function taking a secret key <i>k</i> and a value <i>x</i> as input.  This function is pseudorandom if the keyed function is indistinguishable from a randomly sampled function acting on the same domain and range.  In the KKRT OPRF [1], one party (the sender) holds the PRF secret key, and the other (the receiver) holds the PRF output evaluated using the secret key on his inputs. The sender can later on use the secret key to evaluate the OPRF output on any input. The 'obliviousness' property ensures that the sender does not learn anything about the receiver's input during the evaluation.  The receiver should also not learn anything about the sender's secret PRF key. This can be efficiently implemented by slighly modifying the KKRT <i>1 out of n</i> OT extension protocol.

## References

[1] V. Kolesnikov, R. Kumaresan, M. Rosulek, N.Trieu. Efficient Batched Oblivious PRF with Applications to Private Set Intersection. Full version available at https://eprint.iacr.org/2016/799.pdf, and ACM version at https://dl.acm.org/doi/pdf/10.1145/2976749.2978381.