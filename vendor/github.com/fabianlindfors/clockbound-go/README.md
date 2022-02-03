# clockbound-go

Native Go client for [AWS Clockbound](https://github.com/aws/clock-bound). Requires a running [ClockboundD daemon](https://github.com/aws/clock-bound/blob/main/clock-bound-d/README.md).

## Usage

```go
// Connects to the default /run/clockboundd/clockboundd.sock socket.
// Use clockbound.NewWithPath("/path/to/socket") to use a custom socket path.
clock, err := clockbound.New()
if err != nil {
	panic(err)
}

// Now() returns a bounded timestamp with type `Bounds`
bounds, err := clock.Now()
if err != nil {
	panic(err)
}

// `Bounds` has two fields, Earliest and Latest. They are two timestamps
// setting bounds on the actual current time.
// The timestamps are counted as nanoseconds since the UNIX epoch.
fmt.Printf("Earliest timestamp: %d\n", bounds.Earliest)
fmt.Printf("Latest timestamp: %d\n", bounds.Latest)

// Before and After are convenience methods for checking if a timestamp is
// before or after the current time
// The parameter should be a timestamp of nanoseconds since the UNIX epoch.
before, err := clock.Before(0)
if err != nil {
	panic(err)
}
// If before == true, then the timestamp is before the current time

after, err := clock.After(0)
if err != nil {
	panic(err)
}
// If after == true, then the timestamp is after the current time
```

## License

clockbound-go is released under the [MIT license](LICENSE.md).