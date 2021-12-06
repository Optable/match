# ECDH PSI implementation

## protocol

The Diffie-Hellman private set intersection (DHPSI) [1] is one of the first PSI protocols and is communication efficient, but requires expensive computations from both parties: a sender and a receiver. We implement DHPSI using elliptic curve (specifically `ristretto255` [2]) instead of finite field exponentiation for performance reasons. The point operation of _kP_ is the multiplication of a ristretto point _P_ with a scalar _k_ over an ellipic curve (Curve25519).

1. the receiver and the sender agree on a preset elliptic curve _E_ (Curve25519).
1. the sender generates his private key (*scalar*) _a_, and hashes each identifier from his input audience list to obtains points _x<sub>i</sub> ∈ X_ on _E_. (*Derive*)
1. for each of the derived points _x<sub>i</sub>_, the sender computes point multiplication: _ax<sub>i</sub>_, and obtains the set of multiplied points _aX_. (*Multiply*)
1. the sender permutes _aX_ and sends them to the receiver. (*ShuffleWrite*)
1. the receiver receives _aX_ from the sender. The receiver generates his private key (*scalar*) _b_ and multiplies each element _ax<sub>i</sub> ∈ aX_ with private key _b_: _b(ax<sub>i</sub>)_ to obtain the set of multiplied points _baX_ and index it. (*ReadMultiply*)
1. the receiver hashes each identifier from his input audience list to obtain points _y<sub>i</sub> ∈ Y_ on _E_. (*Derive*)
1. for each of the derived points _y<sub>i</sub>_, the receiver computes point multiplication: _by<sub>i</sub>_, and obtains the set of multiplied points _bY_. (*Multiply*)
1. the receiver permutes _bY_ and sends them to the sender. (*ShuffleWrite*)
1. The sender receives _bY_, and multiplies each element _by<sub>i</sub> ∈ bY_  with his private key _a_: _a(by<sub>i</sub>)_ to obtain the set of multiplied points _abY_, and sends them back to the receiver. (*ReadMultiply*)
1. the receiver receives _abY_, and finds the intersecting identifiers from the set _baX_ and _abY_. (*Intersect*)


## data flow

```
          Sender                                        Receiver
          X, a                                          Y, b


Stage 1   DM/Shuffle    --------------aX------------->  M -> baX              Stage 1

Stage 2   M -> abY      +-------------bY--------------  DM/Shuffle            Stage 2.1
                        |
                        +-------------abY------------>  intersect(baX, abY)   Stage 2.2


     DM:  ristretto255  derive/multiply
      M:  ristretto255  multiply
Shuffle:  cryptographic quality shuffle
```

## References

[1] C. Meadows. A more efficient cryptographic matchmaking protocol for use in the absence of a continuously available third party. In IEEE S&P’86, pages 134–137. IEEE, 1986.

[2] https://datatracker.ietf.org/doc/html/draft-irtf-cfrg-ristretto255-decaf448

