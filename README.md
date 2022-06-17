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

Build client (`s83`) and server (`s83d`).
```
$ go build -o bin/s83 cmd/client/*
$ go build -o bin/s83d cmd/server/*
$ cd bin
```

Make a directory (`store`) for the server to store it's boards, then start server
and leave it running in another window.
```
$ mkdir store
$ ./s83d
```

Generate a creator key ("secret"). This may take a few minutes to get lucky.
```
$ ./s83 new
```

Add the keys you just generated to your client configuration and set the server
to point at your local server like so.

`~/.config/s83/config`
```
public = db8a22f49c7f98690106cc2aaac15201608db185b4ada99b5bf4f222883e1223
secret = XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
server = http://localhost:8080
```

Verify you can reach your server by getting the ever-changing test board.
```
$ ./s83 get ab589f4dde9fce4180fcf42c7b05185b0a02a5d682e353fa39177995083e0583
```

Make a board and publish it to the world!
```
$ echo "<h1>It's Alive</h1>" > board.html

$ ./s83 pub board.html
[info] Success
```

Check out your great work (use _your_ public creator key).
```
$ ./s83 get db8a22f49c7f98690106cc2aaac15201608db185b4ada99b5bf4f222883e1223
```

Enjoy!

## Current Limitations

While the core protocol functionality works, some features still need to be
implemented. A non-exhaustive list of missing features follows.

- server
	- missing gossip
	- missing difficulty verification
	- potentially imprecise error messages
- client
	- no "realms"
	- no list of multiple subscriptions (e.g. get a single board at a time)
	- plaintext only, no DOM/CSP/Formatting
- Tests

Development continues. This list should shrink over time as the spec is refined
and solidifies. The primary purpose at this point is to explore the corners of
spec, provide friendly comment, and have a good time implementing something
cool.

## Design Decisions

- server
  - golang all standard library with minimal dependencies
  - flat/plaintext files for persistent storage
- client
  - golang that spits out an html page and fires off a browser
