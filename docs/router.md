stream router
=============

* Constructs stream plumbing according to configuration
* 3 types:
  * Generators: generate a message stream in some way
  * Filter: receive and generate a message stream
  * Output: receive a message stream and output nothing
* Types can be autodetected via signature and enforced according
  to requested configuration.


## Example

```toml

[TCP]
address = ":8213"

[S3]
key = "blah"

[archive.S3]
type = "S3"
key = "foo"

[Spool]
dir = "/var/spool/cypress"

[giant.Spool]
dir = "/mnt/giant/cypress"

[Statsd]
address = ":33412"

[pipes.Default]
generate = ["TCP", "Statsd"]
output = ["S3", "Spool"]

[pipes.Rare]
enabled = false
generate = ["TCP", "Statsd"]
output = ["archive", "giant"]

```
