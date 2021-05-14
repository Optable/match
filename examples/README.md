# examples

## 1. generate some data
`go run ./generate.go`

This will create two files, `sender-ids.txt` and `receiver-ids.txt` with 100 IDs in common between them. You can confirm the communality by running:

```
comm -12 <(sort sender-ids.txt) <(sort receiver-ids.txt) | wc -l
     100
```

## 2. run the receiver
`go run receiver/main.go`

## 3. start a sender