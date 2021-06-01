# ecdh psi implementation

## protocol

The Elliptic Curve Diffie-Hellman private set intersection. The point operation of _kP_ is multiplication of a point _P_ with a scalar _k_ over an elliptic curve.

1. the receiver and the sender agree on an elliptic curve _E_.
1. the sender generates his private key _a_, and hashes each identifier from his input audience list to obtain points _x<sub>i</sub> ∈ X_ on E. (*Derive*)
1. for each of the derived points _x<sub>i</sub>_, the sender computes point multiplication: _ax<sub>i</sub>_, and obtain points _aX_. (*Multiply*)
1. the sender permutes _aX_ and sends them to the receiver. (*ShuffleWrite*)
1. the receiver receives _aX_ from the sender. The receiver generates his private key _b_ and performs point multiplication on _aX_ with secret key _b_: _b(ax<sub>i</sub>)_, to obtain _baX_ and index it. (*ReadMultiply*)
1. the receiver hashes each identifier from his input audience list to obtain points _y<sub>i</sub> ∈ Y_ on _E_. (*Derive*)
1. for each of the derived points _y<sub>i</sub>_, the receiver computes point multiplication: _by<sub>i</sub>_, and obtain points _bY_. (*Multiply*)
1. the receiver permutes _bY_ and sends them to the sender. (*ShuffleWrite*)
1. The sender receives _bY_, and performs point multiplication on _bY_ with his secret key _a_: _a(by<sub>i</sub>)_, and obtains _abY_, and sends them back to the receiver. (*ReadMultiply*)
1. the receiver receives _abY_, and finds the intersecting identifier from the set _baX_ and _abY_. (*Intersect*)


## data flow

```
         Sender                                        Receiver
         X, a                                          Y, b


Stage1   DM/Shuffle    --------------aX------------->  M -> baX              Stage 1

Stage2   M -> abY      +-------------bY--------------  DM/Shuffle            Stage 2.1
                       |
                       +-------------abY------------>  intersect(baX, abY)   Stage 2.2


     DM: ristretto255  derive/multiply
      M: ristretto255  multiply
Shuffle: cryptographic quality shuffle
```
