# ecdh psi implementation

The Elliptic Curve Diffie-Hellman private set intersection. The point operation of _kP_ is mulitplication of a point _Y_ with a scalar _k_ over an elliptic curve defined over a finite field modulo a prime number.

1. The Publisher and the Advertiser agree on an elliptic curve _E_.
2. The Advertiser generates his private key _a_, and hashes each identifier from his input audience list to obtain points _x<sub>i</sub> ∈ X_ on E. (derive)
3. Then for each of the derived points _x<sub>i</sub>_, the Advertiser computes point multiplication: _ax<sub>i</sub>_, and obtain points _aX_. (multiply)
4. The Advertiser permutes _aX_ and sends them to the Publisher. (shuffle/write remote)
5. The Publihser receives _aX_ from the Advertiser. The Publisher generates his private key _b_ and performs point multiplication on _aX_ with secret key _b_: _b(ax<sub>i</sub>)_, to obtain _baX_. (read remote, multiply)
6. The Publisher hashes each identifier from his input audience list to obtain points _y<sub>i</sub> ∈ Y_ on _E_. (derive)
7. For each of the derived points _y<sub>i</sub>_, the Publisher computes point multiplication: _by<sub>i</sub>_, and obtain points _bY_. (multiply)
8. The Publisher permutes _bY_ and sends them to the Advertiser. (shuffle/write remote)
9. The Advertiser receives _bY_, and performs point multiplication on _bY_ with his secret key _a_: _a(by<sub>i</sub>)_, and obtains _abY_, and sends them back to the Publisher. (read remote, multiply)
10. Finally, the publisher receives _abY_, and finds the intersecting identifier from the set _baX_ and _abY_. (intersect)


# data flow

## sender
    stage 1: (read identifier) -> (derive/multiply,shuffle/write remote)
    stage 2: (read remote,multiply) -> (write remote)
## receiver
    stage 1: (read remote,multiply) -> index
    stage 2: (read identifier) -> (derive/multiply,shuffle/write remote)
    stage 3: (read remote) -> intersect
