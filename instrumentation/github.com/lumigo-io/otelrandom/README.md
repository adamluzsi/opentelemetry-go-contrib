# otelrandom

This package is an example instrumentation for a predefined random function interface
for the sake of Lumigo OTeL engineer exam.

## Specification

Implement payload instrumentation for the following API:

```go
package example

type RandomGenerator interface {
	Intn(ctx context.Context, n int) int
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

- [x] contain the Lumigo span attributes
    - [x] input value as JSON encoded payload
    - [x] the size of that JSON encoded payload
    - [x] process CPU time
- [x] If traceparent is present in the context, it must use it for the newly created tracing span

**It Should**:

- [x] Follow the idioms of other contrib packages
    - [x] provide a factory function with variadic Option parameters
    - [x] Use otel globals in case like TraceProvider is not supplied

**It is Nice To Have** (not requirement, and should only do if we have enough remaining time):

- [ ] provide metrics as well as part of the instrument
- [ ] support Lumigo's payload masking feature
  to some degree through an optional configuration
  or through environment variables

Required span attributes:

- method.payload - JSON structured content of the captured payload
- method.payload.size - Size in bytes of the captured payload
- process.cpu.time

### Task 2

While random generation doesn't involve asynchronous interactions, for the sake of exercise,
the solution should extended with an example where an asynchronous action is made,
this action is traced with a new span, which is linked back to the random generator's span context.
The goal of this task is to showcase how using spans as part of an asynchronous process can be linked back to an
original span.

- [x] Async call started as part of the Intn method call
- [x] Async call has its own span context
    - [x] the traceparent is the outer trace not the span created as part of the Intn method
    - [x] span created for Intn method is linked in the async call's span

## Architecture Decision Records

### Flat UnitTest Testing Style

For this instrumentation package, which is intended for open source contribution,
I'll employ a flat unit testing style given its widespread familiarity.
In the event that this package becomes exclusively maintained by Lumigo in the future,
we could consider transitioning to a more maintainable, modular nested testing style.

### process.cpu.time

The label "process.cpu.time" might seem unclear without a precise context.
I've interpreted it in alignment with other attributes that use the "process." prefix.

Obtaining this data typically necessitates either active pprof usage or system interaction via syscalls.
To ensure portability and to avoid a platform-specific implementation,
I've opted to leverage the gopsutil open-source library to fetch the CPU time.

### Async Span Example

Given that a random number generator seldom requires asynchronous operations,
I've kept the implementation straightforward.
The design primarily illustrates span creation in an asynchronous context
and demonstrates how to link back to another span.
Were the use-case to involve more intricate scenarios,
like tracing asynchronous calls across multiple backend servers,
the example would be more intricate and engaging.

### Structure and API Design

The design of this instrument package closely aligns with the conventions seen in otel's instrument packages.
For instance, it employs an unexported configuration type that encapsulates the instrument's setup.
This can be configured through the Factory function' (NewRandomGenerator) variadic Option parameter.

```go
otelrandom.NewRandomGenerator(randomGenerator,
    otelrandom.WithTracerProvider(MyTracerProvider))
```

While this might not be an idiomatic Go approach, it does increase the likelihood of our contribution being accepted,
given its similarity in look and feel to other instrument packages.

A more idiomatic Go approach might involve using a struct with dependencies presented as exported fields.
This structure allows for straightforward dependency injection by populating these fields, streamlining testing.
If dependencies aren't provided, the system defaults to zero values,
like the otel global values (e.g., otel.GetTracerProvider()).
