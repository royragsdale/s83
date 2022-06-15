# s83

A sample [Spring '83](https://www.robinsloan.com/lab/specifying-spring-83/)
client and server implementation according to the draft
[spec](https://github.com/robinsloan/spring-83-spec/blob/main/draft-20220609.md)
in response to Robin’s Request For Friendly Critique and Comment (RRFFCC).

**This is fun. This will change**

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
2022/06/15 08:16:01 loaded 0 boards from store [...]
2022/06/15 08:16:01 server started on :8080
```

Generate a creator key ("secret"). This may take a few minutes to get lucky.
```
$ ./s83 new
[info] Config did not exist. Initializing a config at [...]
[info] Success! Found a valid key in 9420739 iterations over 226 seconds (41558 kps)
[info] The public key is your creator id. Share it!
[WARN] The secret key is SECRET. Do not share it or lose it.
public: 9e2d995d90c1a473809eb259a6ad578faf111b234f868f143caeec6cc5ed2021
secret: XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
```

Add the keys you just generated to your client configuration and set the server
to point at your local server like so.

`~/.config/s83/config`
```
public = 9e2d995d90c1a473809eb259a6ad578faf111b234f868f143caeec6cc5ed2021
secret = XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
server = http://localhost:8080
```

Verify you can reach the server by getting the ever-changing test board.
```
$ ./s83 get fad415fbaa0339c4fd372d8287e50f67905321ccfd9c43fa4c20ac40afed1983
verifies  : true
creator   : fad415fbaa0339c4fd372d8287e50f67905321ccfd9c43fa4c20ac40afed1983
signature : 0b70a31798a38ffb0c81299f7af0335af18fc4c80db71a657ba9bb5d857efae93c260a4843029c7fe1edbe68b8e62712f5f8f844e770aa3d592b4c221b27e60d
<meta http-equiv="last-modified" content="Wed, 15 Jun 2022 12:18:35 GMT"><!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>s83d | Hello World</title>
</head>
<body>
  <h1>Magic s83-ball</h1>
  <p>Better not tell you now.</p>
</body>
</html>
```

Make a board and publish it to the world!
```
$ echo "<h1>It's Alive</h1>" > board.html

$ ./s83 pub board.html
[info] Success
```

Check out your great work (use _your_ public creator key).
```
$ ./s83 get 9e2d995d90c1a473809eb259a6ad578faf111b234f868f143caeec6cc5ed2021
verifies  : true
creator   : 9e2d995d90c1a473809eb259a6ad578faf111b234f868f143caeec6cc5ed2021
signature
: 6f82bacced04693c811f5ec689b8f354219b59b25b14b40409cb5150e57299e943cbcdc8066fb13fc94804dc1288ad2add637bbfdb8aa30c8b3ce6cb1f49e409
<meta http-equiv="last-modified" content="Wed, 15 Jun 2022 12:40:01
GMT"><h1>It's Alive</h1>
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
spec and provide friendly comment.

## Design Decisions

- server
  - golang all standard library with minimal dependencies
  - flat/plaintext files for persistent storage
- client
  - golang that spits out an html page and fires off a browser
