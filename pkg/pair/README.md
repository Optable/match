# Publisher Advertiser Identity Reconciliation (PAIR)

## Overview
Publisher Advertiser Identity Reconciliation (PAIR) [1] is a privacy-centric protocol that leverages commutative encryptions like the diffie-hellman PSI to reconcile the first party identitfiers of the publisher and the advertiser, and allows a secure programmatic activation for the advertiser following a PAIR match.

This package provides a reference implementation of core utility functions required to run the PAIR protocol.

## Protocol
The PAIR protocol in the dual clean room scenario involves two clean room operators, one responsible for the publisher and the other for the advertiser. The protocol consists of the following steps:

### Key generation and management
1. the publisher clean room operator __P__ and the advertiser clean room operator __A__ agree on a hashing function (SHA256) and a preset elliptic curve _E_ (Curve25519).
2. __P__ generates a random hash salt _s_, and a private key (*scalar*) _pSK_, and rotates them periodically. _s_ is rotated every 30 days, and _pSK_ is rotated every 180 days.
2. __A__ generates a private key _aSK_, and rotates it every 180 days.

### offline matching
1. __P__ shares the hash salt _s_ with __A__.
2. __P__ hashes each identifier _x<sub>i</sub> ∈ X_ from his input audience list _X_ and encrypts the hashed identifiers using _pSK_ to obtain _E<sub>pSK</sub>(H<sub>s</sub>(x<sub>i</sub>))_, which is also known as the Publisher ID.
3. __A__ hashes each identifier _y<sub>i</sub> ∈ Y_ from his input audience list _Y_ and encrypts the hashed identifiers using _aSK_ to obtain _E<sub>aSK</sub>(H<sub>s</sub>(y<sub>i</sub>))_, which is also known as the Advertiser ID.
4. __P__ and __A__ exchange the Publisher IDs and Advertiser IDs respectively.
5. __P__ encrypts the Advertiser IDs using _pSK_ to obtain _E<sub>pSK</sub>(E<sub>aSK</sub>(H<sub>s</sub>(y<sub>i</sub>)))_, the doubly encrypted and hashed identifier is known as the PAIR ID.
6. __A__ encrypts the Publisher IDs using _aSK_ to obtain _E<sub>aSK</sub>(E<sub>pSK</sub>(H<sub>s</sub>(x<sub>i</sub>)))_, which is known as the PAIR ID.
7. __P__ and __A__ exchange the PAIR IDs respectively.
8. __P__ intersects the PAIR IDs to obtain the match rate, and output a table containing his un-encrypted identitifiers _x<sub>i</sub>_ and its Publisher ID counter part _E<sub>pSK</sub>(H<sub>s</sub>(x<sub>i</sub>))_.
9. __A__ intersects the PAIR IDs to obtain the match rate, and the intersected PAIR IDs. __A__ decrypts the PAIR IDs using _aSK_ to obtain the intersected Publisher IDs _E<sub>pSK</sub>(H<sub>s</sub>(y<sub>i</sub>))_.

### online activation
1. __A__ sends the intersected Publisher IDs to his Demand Side Platform (DSP) for activation.
2. __P__ keeps the mapping of his identifier _x<sub>i</sub>_ and the Publisher ID _E<sub>pSK</sub>(H<sub>s</sub>(x<sub>i</sub>))_.
3. When a user visits the publisher's website, the publisher looks up the Publisher ID of the visitor and prepares an OpenRTB bid request containing the Publisher ID to his Sell Side Platform (SSP).
4. The SSP sends the bid request to the DSP.
5. The DSP looks up the Publisher ID in the list of intersected Publisher ID sent by __A__ and decides the outcome of the bid request.

## References

[1] https://docs.google.com/document/d/1sPw1MT0NxO3xTI4Rp5zyA1G7eAtOJBpgj5x5x87OP-w/edit?usp=sharing
