# examples

The standard match operation involves a *sender* and a *receiver*. The sender performs an intersection match with a receiver, such that the receiver learns the result of the intersection, and the sender learns nothing. Protocols such as PSI allow the sender and receiver to  protect, to varying degrees of security guarantees and without a trusted third-party, private data records that are used as inputs in performing the intersection match.

## 1. generate some data
`go run generate.go`

This will create two files, `sender-ids.txt` and `receiver-ids.txt` with 100 *IDs* in common between them. You can confirm the communality by running:

`comm -12 <(sort sender-ids.txt) <(sort receiver-ids.txt) | wc -l`

## 2. run the receiver
`go run receiver/main.go`

The receiver will learn of the intersection between `sender-ids.txt` and `receiver-ids.txt` and write the results to `common-ids.txt`

## 3. start a sender
`go run sender/main.go`

The sender sends the contents of `sender-ids.txt` to the receiver but learns nothing.

## 4. verify the intersection
```
comm -12 <(sort receiver-ids.txt) <(sort common-ids.txt) | wc -l
comm -12 <(sort sender-ids.txt) <(sort common-ids.txt) | wc -l
```

