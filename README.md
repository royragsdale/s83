# s83

A sample [Spring '83](https://www.robinsloan.com/lab/specifying-spring-83/)
client and server implementation according to the draft
[spec (16 JUN)](https://github.com/robinsloan/spring-83-spec/blob/main/draft-20220616.md)
in response to Robin’s Request For Friendly Critique and Comment (RRFFCC) and
for fun.

**Very much in flux**, but sufficiently conformant to publish and get from the
demo server!

I'm springing at `db8a22f49c7f98690106cc2aaac15201608db185b4ada99b5bf4f222883e1223`.

## Quick Start/Demo

1. Build client (`s83`) and server (`s83d`).
```
$ go build -o bin/s83 ./cmd/client
$ go build -o bin/s83d ./cmd/server
$ cd bin
```

2. Make a directory (`store`) for the server to store it's boards, then start server
and leave it running in another window.
```
$ mkdir store
$ ./s83d
```

3. Generate a creator key ("secret"). This may take a few minutes to get lucky.
   Speed it up with (`-j N`) where `N` is the number of miners to run. Locally
   I get ~160k attempts per second.
```
$ ./s83 new
```

4. Add your keys to a "profile"

_You can have multiple profiles and switch between them with `./s38 -c PROFILE`_

Add the keys you just generated to your client profile and set the server to
point at your local server like below.

Aside from the simple `key = value` lines this attempts to conform to the
"Springfile" convention so you can add a list of the boards you would like to
follow, optionally preceded by a _handle_ for how you would like to track that
board (`me` in this example).

`~/.config/s83/default`
```
secret = XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
server = http://localhost:8080

me
http://localhost:8080/db8a22f49c7f98690106cc2aaac15201608db185b4ada99b5bf4f222883e1223
```

5. Verify you can reach your server by getting the ever-changing test board.
```
$ ./s83 get ab589f4dde9fce4180fcf42c7b05185b0a02a5d682e353fa39177995083e0583
```

6. Make a board and publish it (don't worry this is just to your local test
   server)!
```
$ echo "<h1>It's Alive</h1>" > board.html

$ ./s83 pub board.html
[info] Success
```

7. Assuming you added your key to your configuration you can check out your great
work with a simple `get` which will fetch all of your configured/followed boards.
```
$ ./s83 get
```

At this point the client only fetches the raw boards, so you can list/view them
however is convenient:
```
$ ls ~/.config/s83/data/default/*
$ cat ~/.config/s83/data/default/*
```

Enjoy!

Some public servers you can publish boards to are:
- [https://bogbody.biz](https://bogbody.biz)
- [https://0l0.lol/](https://0l0.lol/)

## Current Limitations

While the core protocol functionality works, some features still need to be
implemented. A non-exhaustive list of missing features follows.

- server
	- missing gossip
	- missing difficulty verification
	- potentially imprecise error messages
- client
	- no "realms"
	- plaintext only, no DOM/CSP/Formatting
- Tests

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

## Design Decisions

- server
  - golang all standard library with minimal dependencies
  - flat/plaintext files for persistent storage
- client
  - golang that spits out an html page and fires off a browser
