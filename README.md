# High Frequency Monitor (hfm)

hfm is a go server application to run tests in parallel at a high frequency.
If the outcome of the test results in a state change, other commands can be
executed.

It is designed to be a general purpose tool, by having both the tests and
the state change commands be interpereted by a shell, such as /bin/sh.

An example application is to poll other network services for health, and
to take actions based on their health status changes.

## Building

There's a patch-local-go-libucl make target that will allow you to use
the locally installed libucl vs. a vendorized version
