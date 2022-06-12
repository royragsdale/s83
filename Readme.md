# s83

A sample [Spring '83](https://www.robinsloan.com/lab/specifying-spring-83/)
client and server implementation according to the draft
[spec](https://github.com/robinsloan/spring-83-spec/blob/main/draft-20220609.md).

## Design Decisions

- server
  - golang all standard library with minimal dependencies
  - flat files
- client
  - golang that spits out an html page and fires off a browser

## Feedback

Some notes/thoughts while implementing the spec.

### Difficulty calculation

The difficulty calculation is a bit sharp edged with fractional math over big
numbers. It is unclear the level of precision that is expected.  Perhaps
a solution would be something that doesn't involve floats. Maybe just plain
integer/modular division the number of `1` bits in the key.

`numBoards * numbits`/`maxBoards` -> {0-numbits}, then count that many `1` bits

I think that has the same general properties.

### Syncing

Corner case with syncing within a realm for servers with different difficulty
levels. If a server is new to a realm and has not yet synced then it will have
very few boards stored, and a very low difficulty factor. Which would allow keys
to be accepted there that would not be accepted at other hosts in the realm.

When these boards are synced, are they accepted at other hosts? It would seem
not.

Another point of divergence (where things are accepted one place and denied
another) would be servers with different block lists. This is potentially
exacerbated by the `respond-async` with deferred validation.

It's unclear on the "peer to peer PUT /<key>" if that is special cased or if it
is intended to be the same logic as the client->server PUT.

It seems that the difference between client->server server->server is not
especially distinguishable. And even if it was there is no mention of the server
validating that another server is in it's "realm" before accepting puts (perhaps
a server defined behavior). 
