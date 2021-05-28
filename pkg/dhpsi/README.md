# dhpsi implementation

This is an implementation of the DHPSI matching protocol.

(i) The Publisher computes hP = H(ui)r1 of each user identifier ui in its audience list, where H is a hash function and r1 is chosen uniformly at random. Let LP be the list of all such hP.


## sender
    stage 1: (read identifier) -> (derive/multiply,shuffle/write remote)
    stage 2: (read remote,multiply) -> (write remote)
## receiver
    stage 1: (read remote,multiply) -> index
    stage 2: (read identifier) -> (derive/multiply,shuffle/write remote)
    stage 3: (read remote) -> intersect
