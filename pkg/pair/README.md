# Publisher Advertiser Identity Reconciliation (PAIR)

## Overview
Publisher Advertiser Identity Reconciliation (PAIR) [1] is a privacy-centric protocol that leverages commutative encryptions like the [Diffie-Hellman](https://github.com/Optable/match/blob/main/pkg/dhpsi/README.md) PSI to reconcile the first party identitfiers of the publisher and the advertiser, and allows a secure programmatic activation for the advertiser following a PAIR match.

This package provides a reference implementation of core utility functions required to run the PAIR protocol.

## Protocol
The PAIR protocol in the dual clean room scenario involves two clean room operators, one responsible for the publisher and the other for the advertiser. The protocol consists of the following steps:

### Key generation and management
1. the publisher clean room operator __Pub__ and the advertiser clean room operator __Adv__ agree on a hashing function (SHA256) and a preset elliptic curve (Curve25519).
2. __Pub__ generates a random hash salt _s_, and a private key (*scalar*) _p_, and rotates them periodically. _s_ is rotated every 30 days, and _p_ is rotated every 180 days.
2. __Adv__ generates a private key _a_, and rotates it every 180 days.

### offline matching
1. __Pub__ shares the hash salt _s_ with __Adv__.
2. __Pub__ hashes each identifier _x<sub>i</sub> ∈ X_ from his input audience list _X_ and encrypts the hashed identifiers using _p_ to obtain _E<sub>p</sub>(H<sub>s</sub>(x<sub>i</sub>))_, which is also known as the Publisher ID.
3. __Adv__ hashes each identifier _y<sub>i</sub> ∈ Y_ from his input audience list _Y_ and encrypts the hashed identifiers using _a_ to obtain _E<sub>a</sub>(H<sub>s</sub>(y<sub>i</sub>))_, which is also known as the Advertiser ID.
4. __Pub__ and __Adv__ exchange the Publisher IDs and Advertiser IDs respectively.
5. __Pub__ encrypts the Advertiser IDs using _p_ to obtain _E<sub>p</sub>(E<sub>a</sub>(H<sub>s</sub>(y<sub>i</sub>)))_, the doubly encrypted and hashed identifier is known as the PAIR ID.
6. __Adv__ encrypts the Publisher IDs using _a_ to obtain _E<sub>a</sub>(E<sub>p</sub>(H<sub>s</sub>(x<sub>i</sub>)))_, which is known as the PAIR ID.
7. __Pub__ and __Adv__ exchange the PAIR IDs respectively.
8. __Pub__ intersects the PAIR IDs to obtain the match rate, and output a table containing his un-encrypted identitifiers _x<sub>i</sub>_ and its Publisher ID counter part _E<sub>p</sub>(H<sub>s</sub>(x<sub>i</sub>))_.
9. __Adv__ intersects the PAIR IDs to obtain the match rate, and the intersected PAIR IDs. __Adv__ decrypts the PAIR IDs using _a_ to obtain the intersected Publisher IDs _E<sub>p</sub>(H<sub>s</sub>(y<sub>i</sub>))_.

### online activation
1. __Adv__ sends the intersected Publisher IDs to his Demand Side Platform (DSP) for activation.
2. __Pub__ keeps the mapping of his identifier _x<sub>i</sub>_ and the Publisher ID _E<sub>p</sub>(H<sub>s</sub>(x<sub>i</sub>))_.
3. When a user visits the publisher's website, the publisher looks up the Publisher ID of the visitor and prepares an OpenRTB bid request containing the Publisher ID to his Sell Side Platform (SSP).
4. The SSP sends the bid request to the DSP.
5. The DSP looks up the Publisher ID in the list of intersected Publisher ID sent by __Adv__ and decides the outcome of the bid request.

## References
[1] TBD.
