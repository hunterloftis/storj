# Overview

Based on the [Storj problem](problem.md):

Terminal 1:
```
$ ./relay :9021
```

Terminal 2:
```
$ ./send localhost:9021 test/olivia.jpg
little-earth-music
```

Terminal 3:
```
$ ./receive localhost:9021 little-earth-music test2/
$ diff test/olivia.jpg test2/olivia.jpg
```

I thought this was a really interesting challenge so I hacked a version together after reading about it [on Reddit](https://www.reddit.com/r/golang/comments/eyphsm/golang_homework_interview_challenge_for_storj/).
Given that they plan to [replace the now-public challenge](https://www.reddit.com/r/golang/comments/eyphsm/golang_homework_interview_challenge_for_storj/fgixfb3/), it doesn't seem like I'm spoiling anything.
That said, Storj, please reach out if you'd rahter this not be on GitHub.

# Design

This solution places a listening HTTP/2 server at the relay host.
The relay server accepts requests to stream a file on `/send`,
where it generates and responds with a secure code before pausing transmission while waiting on a receiver.
Receivers request files by passing the secure code to the `/receive` endpoint.

## Considerations

A solution could be built with just TCP, rather than on top of HTTP.
However, that solution would need to invent some sort of protocol for indicating "sending," "receiving,"
"ready to receive," "proposed filename," and so forth. HTTP provides well-understood mechanisms for this.

The server enforces Transport Layer Security (TLS) for the privacy and data integrity of transfers.
For simplicity, a self-signed cert is hardcoded into the single `relay` binary and `InsecureSkipVerify`
is set on the clients, but a production deployment would use proper certificates.

HTTP/2 allows `/send` to be multiplexed, so sending can be implemented as one simple function.
With HTTP/1.1's request-response, we'd need to implement something like `/offer` (to get a secure code) and `/send` (to stream the file),
and then coordinate between them.

To make secret codes easier to share via voice,
they're generated by randomly selecting three words in sequence from a dictionary of the most-common
800-or-so English words (eg, `little-earth-music`). This provides a search space of a little over half-a-billion
strings, which seems reasonable for this situation.
To provide additional security, the relay server could either implement, or be placed behind,
a rate-limiting service to prevent brute-force attacks over the network.

Although it wasn't in the spec, a future improvement to consider is timeouts for the Send and Receive routes.
As-is, a bad (or mistaken) actor could hold open resources by making many concurrent connections.

# Local development

## Testing

```
$ go test ./...
```

## Building

```
$ make
$ cd dist
```