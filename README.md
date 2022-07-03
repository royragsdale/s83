# s83

A sample [Spring '83](https://www.robinsloan.com/lab/specifying-spring-83/)
client and server implementation according to the draft
[spec (29 JUN)](https://github.com/robinsloan/spring-83-spec/blob/main/draft-20220629.md)
in response to Robin’s Request For Friendly Critique and Comment (RRFFCC) and
for fun.

**Very much in flux**, but sufficiently conformant to publish and get from the
demo server!

Hosted server at [may83.club](https://may83.club).

Publishing at [db8a22f49c7f98690106cc2aaac15201608db185b4ada99b5bf4f222883e1223](https://may83.club/db8a22f49c7f98690106cc2aaac15201608db185b4ada99b5bf4f222883e1223).

## Quick Start/Demo

#### 1. Build client (`s83`)
```
$ go build -o bin/s83 ./cmd/client
$ cd bin
$ ./s83 --help
```

#### 2. Generate a creator key ("secret").

This may take a few minutes to get lucky.

```
$ ./s83 new
```

Speed it up with (`-j N`) where `N` is the number of miners to run. Locally
I get ~160k attempts per second.


#### 3. Add your keys to a "profile"

Add the keys you just generated to the `default` client profile.

`~/.config/s83/default`
```
secret = XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
server = https://may83.club

me
http://may83.club/db8a22f49c7f98690106cc2aaac15201608db185b4ada99b5bf4f222883e1223
```

_You can have multiple profiles and switch between them with `./s38 -c PROFILE`_

Aside from the simple `key = value` lines this attempts to conform to the
"Springfile" convention. You can add a list of the board URLs you would like to
follow, optionally preceded by a _handle_ for how you would like to track that
board (`me` in the example above).

#### 4. Verify you can reach the server by getting the ever-changing test board.
```
$ ./s83 get ab589f4dde9fce4180fcf42c7b05185b0a02a5d682e353fa39177995083e0583
```

#### 5. Make a board and publish it.
```
$ echo "<h1>It's Alive</h1>" > board.html

$ ./s83 pub board.html
[info] Success
```

#### 6. Get your board.

Assuming you added your key to your configuration you can check out your great
work with a simple `get`. This will fetch all of your/followed boards, and
render them to a local HTML file, _The Daily Spring_.

```
$ ./s83 get
```

You just distilled the distributed, ephemeral Springiverse into a single, fully
encapsulated, immutable, cryptographically verified, personal periodical, _The
Daily Spring_.

This is now just a local file. You can save it for later, open it right up in
your browser of choice, email it to yourself or a friend, really anything.

There are some helpful options like `-go`, which immediately opens a browser for
instant gratification, `-new` which will only show you boards you haven't seen,
and `-o` to save the output to a specific path. For example:

```
$ ./s83 get -go -o the-daily-spring.html
```

#### 7. Enjoy!

In addition to [https://may83.club](https://may83.club). Some other public
servers you can publish/follow boards to/from are:

- [https://bogbody.biz](https://bogbody.biz)
- [https://0l0.lol/](https://0l0.lol/)
- [https://spring83.kindrobot.ca](https://spring83.kindrobot.ca)
- [https://spring83.mozz.us](https://spring83.mozz.us/)


## Current Limitations

While the core protocol functionality works, some features still need to be
implemented. A non-exhaustive list of missing features follows.

- server
	- missing gossip
	- potentially imprecise error messages
- client
	- no "realms"
- Tests
- Linux only

Development continues. This list should shrink over time as the spec is refined
and solidifies. The primary purpose at this point is to explore the corners of
spec, provide friendly comment, and have a good time implementing something
cool.

## WIP: Deployment

**work in progress**

There is a [Dockerfile](Dockerfile.server) provided for packaging up the server.
Once things reach a level of stability/release this will get published, but for
now you can build/test locally.

The `Makefile` provides some convenience functions` to spin up an ephemeral test
instance.

```
make docker-build
make docker-serve
```

You can configure the server by setting enviornment variables:

```
$ ./s83d -h
Usage: s83d is designed to be configured using environment variables.

For example: `PORT=8383 ./s83d`

variable         default
--------         -------
HOST
PORT             8080
STORE            store
TTL              22
TITLE            s83d
ADMIN_BOARD
```

### Local Quick Serve

```
make serve
```

## Design Decisions

- server
  - golang all standard library with minimal dependencies
  - flat/plaintext files for persistent storage
- client
  - golang that spits out an html page and fires off a browser
