# otelrandom

This package is an example instrumentation for a predefined random function interface 
for the sake of Lumigo OTeL engineer exam.

## Specification

Implement payload instrumentation for the following API:

```go
type RandomGenerator interface {
    Intn(ctx context.Context, n int) (int, error)	
}
```

This API stands for a function that generates random integer values, and it can also propagate context. 
It's akin to the http.Handler interface but tailored for random number generation. 
The specifics of how the random generator operates aren't our concern here.

The language of choice for this task is Go. 
As Lumigo hasn't begun contributions to the Go contrib repositories,
we'll use the official Go OpenTelemetry contrib repository as our foundation.

**General Notes**
- Use a branch in your local fork of the repository for the instrumentation
- Commits to the branch describe the changes accurately
- Expected Trace structure
  - Root span - the main process
  - Instrumented method span
    - Contains payload attributes
    - create an asynchronously linked sub span
    - Span Link to “Instrumented method span”
- Email the results to dori@lumigo.io and kenf@lumigo.io with a link to your repository fork and branch.

### Task 1

Create a conrib library that helps instrumenting the RandomGenerator API.

**It Must**:
- [ ] contain the Lumigo span attributes
  - [ ] input value as JSON encoded payload 
  - [ ] the size of that JSON encoded payload
- [ ] contain some example semconv use, like the CPU time
- [ ] If traceparent is present in the context, it must use it for the newly created tracing span

**It Should**:
- [ ] Follow the idioms of other contrib packages
  - [ ] provide a factory function with variadic Option parameters
  - [ ] Use otel globals in case like TraceProvider is not supplied

**It is Nice To Have** (not requirement, and should only do if we have enough remaining time):
- [ ] provide metrics as well as part of the instrument

Required span attributes:
- method.payload - JSON structured content of the captured payload
- method.payload.size - Size in bytes of the captured payload
- process.cpu.time

### Task 2

While random generation doesn't involve asynchronous interactions, for the sake of exercise,
the solution should extended with an example where an asynchronous action is made,
this action is traced with a new span, which is linked back to the random generator's span context.
The goal of this task is to showcase how using spans as part of an asynchronous process can be linked back to an original span.

- [ ] Async call started as part of the Intn method call
- [ ] Async call has its own span context
  - [ ] the traceparent is the outer trace not the span created as part of the Intn method
  - [ ] span created for Intn method is linked in the async call's span
