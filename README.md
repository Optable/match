# match
[![match CI](https://github.com/Optable/match/actions/workflows/match-ci.yml/badge.svg?branch=main)](https://github.com/Optable/match/actions/workflows/match-ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/optable/match)](https://goreportcard.com/report/github.com/optable/match)
[![GoDoc](https://godoc.org/github.com/optable/match?status.svg)](https://godoc.org/github.com/optable/match)

An open-source set intersection protocols library written in golang.

The goal of the match library is to provide production level implementations of various set intersection protocols. Protocols will typically tradeoff security for performance. For example, a private set intersection (PSI) protocol provides cryptographic guarantees to participants concerning their private and non-intersecting data records, and is suitable for scenarios where participants trust each other to be honest in adhering to the protocol, but still want to protect their private data while performing the intersection operation.

The standard match operation under consideration involves a *sender* and a *receiver*. The sender performs an intersection match with a receiver, such that the receiver learns the result of the intersection, and the sender learns nothing. Protocols such as PSI allow the sender and receiver to protect, to varying degrees of security guarantees and without a trusted third-party, the private data records that are used as inputs in performing the intersection match.

The protocols that are currently provided by the match library are listed below, along with an overview of their characteristics.

## dhpsi

Diffie-Hellman based PSI (DH-based PSI) is an implementation of private set intersection. It provides strong protections to participants regarding their non-intersecting data records. Documentation located [here](pkg/dhpsi/README.md).

## npsi

The naive, [MetroHash](http://www.jandrewrogers.com/2015/05/27/metrohash/) based PSI: an *insecure* but fast solution for PSI. Documentation located [here](pkg/npsi/README.md).

## bpsi

The [bloomfilter](https://en.wikipedia.org/wiki/Bloom_filter) based PSI: an *insecure* but fast with lower communication overhead than [npsi](pkg/npsi/README.md) solution for PSI. Documentation located [here](pkg/bpsi/README.md).

## kkrtpsi

Similar to the dhpsi protocol, the KKRT PSI, also known as the Batched-OPRF PSI, is a semi-honest secure PSI protocol that has significantly less computation cost, but requires more network communication. An extensive description of the protocol is available [here](pkg/kkrtpsi/README.md).

## logging

[logr](https://github.com/go-logr/logr) is used internally for logging, which accepts a `logr.Logger` object. See the [documentation](https://github.com/go-logr/logr#implementations-non-exhaustive) on `logr` for various concrete implementations of logging api. Example implementation of match sender and receiver uses [stdr](https://github.com/go-logr/stdr) which logs to `os.Stderr`.

### pass logger to sender or receiver
To pass a logger to a sender or a receiver, create a new context with the parent context and `logr.Logger` object as follows
```golang
// create sender and logger
...
// pass logger to context
ctx := logr.NewContext(parentCtx, logger)
err := sender.Send(ctx, n, identifiers)
if err != nil {
    logger.Error(err, "sender failed to send")
}
```
Similarly for receiver,
```golang
// create receiver and logger
...
// pass logger to context
ctx := logr.NewContext(parentCtx, logger)
intersection, err := receiver.Intersect(ctx, n, identifiers)
if err != nil {
    logger.Error(err, "receiver intersection failed")
}
```

### verbosity
Each PSI implementation logs the stage progression, set `logr.Logger` verbosity to `1` to see the logs.
example for `github.com/go-logr/stdr`:
```golang
logger := stdr.New(nil)
stdr.SetVerbosity(1)
```
running the example implementation for `dhpsi` we see the following logs:
```bash
$go run examples/sender/main.go -proto dhpsi -v 1
...
2021/11/11 11:16:12 "level"=1 "msg"="Starting stage 1" "protocol"="dhpsi"
2021/11/11 11:16:12 "level"=1 "msg"="Finished stage 1" "protocol"="dhpsi"
2021/11/11 11:16:12 "level"=1 "msg"="Starting stage 2" "protocol"="dhpsi"
2021/11/11 11:16:12 "level"=1 "msg"="Finished stage 2" "protocol"="dhpsi"
2021/11/11 11:16:12 "level"=1 "msg"="sender finished" "protocol"="dhpsi"
...
```

# testing

A complete test suite for all PSIs is present [here](test/psi). Don't hesitate to take a look and help us improve the quality of the testing by reporting problems and observations! The PSIs have only been tested on **x86-64**.

# benchmarks

See runtime benchmarks of the different PSI protocols [here](benchmark/README.md).

# examples

You can find a simple example implementation of both a match sender and receiver in the [examples documentation](examples/README.md).
