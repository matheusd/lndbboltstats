# LND BBolt DB Stats Gatherer

Runs through a lnd/dcrlnd bbolt db and gathers statistics about size used for
the top-level buckets.

Usage:

```
$ go install github.com/matheusd/lndbboltstats@latest
$ lndbboltstats ~/.dcrlnd/data/graph/mainnet/channel.db
```
