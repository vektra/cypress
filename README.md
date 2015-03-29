<img align="right" src="https://s3-us-west-1.amazonaws.com/assets.vektra.io/images/cypress.png">

cypress
=======

## Toolkit

Cypress is a logging toolkit for creating streams of messages and transporting them.

It's native streams are compressed protocol buffer: reducing bandwidth and reimproving the fidelity of log data.

The toolkit comes with an array of handlers, capable sending streams across networks, saving them to disk, filtering them, and translating them into external formats.

All translations are bidirectional. So if you save a stream into s3, cypress can read that stream back natively. This extends to it's integration with SaaS logging products as well.

The aim is a flexible tool that gives users the highlevel building blocks to create a logging solution that works for them.

### More than logs

Cypress handles metrics as well as logs. In fact, there are 5 messages types:

* **Log** - A log message, representing arbitrary information
* **Metric** - A metric as you'd find in statsd, etc
* **Trace** - An application trace use to correlate activity
* **Audit** - A high value log message
* **Heartbeat** - A simple indicator of aliveness

The types allow the different handlers to create messages specially. For instance,
the builtin `metrics` handler only considers messages of type `Metric`, as you'd
expect.

## Golang

Cypress is a logging framework in golang.

The core is most handlers are written in golang, allowing for extremely easily
deployment.

# Config

Cypress uses configuration files to control some of the various
aspects of it's operation. The format is TOML and here is an
example:

```toml
[s3]

allow_unsigned = true
sign_key = "evan@vektra.com"
```
