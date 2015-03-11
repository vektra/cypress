cypress
=======

Cypress is a logging framework in golang.

It provides a bunch of tooling to create and transport structured messages
representing log entries, metrics, traces, and audits.


### Config

Cypress uses configuration files to control some of the various
aspects of it's operation. The format is TOML and here is an
example:

```toml
[s3]

allow_unsigned = true
sign_key = "evan@vektra.com"
```
