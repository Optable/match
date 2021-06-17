# match
An open-source set intersection protocols library written in golang.

The goal of the match library is to provide production level implementations of various set intersection protocols. Protocols will typically tradeoff security for performance. For example, a private set intersection (PSI) protocol provides cryptographic guarantees to participants concerning their private and non-intersecting data records, and is suitable for scenarios where participants trust each other to be honest in adhering to the protocol, but still want to protect their private data while performing the intersection operation.

The standard match operation under consideration involves a *sender* and a *receiver*. The sender performs an intersection match with a receiver, such that the receiver learns the result of the intersection, and the sender learns nothing. Protocols such as PSI allow the sender and the receiver to protect, to varying degrees of security guarantees and without a trusted third-party, the private data records that are used as inputs in performing the intersection match.

The protocols that are currently provided by the match library are listed below, along with an overview of their characteristics.

## dhpsi

Diffie-Hellman based PSI (DH-based PSI) is an implementation of private set intersection. It provides strong protections to participants regarding their non-intersecting data records. See documentation [here](pkg/dhpsi/README.md).

## npsi

The naive, [highway hash](https://github.com/google/highwayhash) based PSI: an *insecure* but fast solution for PSI. Located [here](pkg/npsi/README.md).

# examples

You can find a simple example implementation of both a match sender and receiver in the [examples documentation](examples/README.md).
