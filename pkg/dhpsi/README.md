# dhpsi implementation

The sequence for encoding and decoding is

## sender
    stage 1: (read identifier) -> (derive/multiply,shuffle/write remote)
    stage 2: (read remote,multiply) -> (write remote)
## receiver
    stage 1: (read remote,multiply) -> index
    stage 2: (read identifier) -> (derive/multiply,shuffle/write remote)
    stage 3: (read remote) -> intersect
