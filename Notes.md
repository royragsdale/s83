# Notes/Feedback

Some scratch/notes/thoughts while implementing the spec...

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

Another edge case is a client can publish different boards with exactly the same
time to multiple servers. This is a divergent state.

### Time

You can't make boards in the future but it seems that you can make boards valid
arbitrarily in the past.

### Expiration and Transitions

Seems like everyone is on a cliff where at the calendar year a large number of
keys will expire/rotate.

Also there seems to be very little support for carrying trust between keys. For
example for a slow client, that doesn't check often, if they miss January, then
they miss the entire past (TTL).

Mechanism for transferring ownership over time.

Really there is no identity.

### Client state

What state is a client required to store, e.g. "
