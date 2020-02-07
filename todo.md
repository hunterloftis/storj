# TODO

1. Figure out why sending is do damned slow on the real Internet (or even LAN).
2. Clean up broken sends (if client disconnects, "sending file: client disconnected") the client can "resume" but nothing ever happens since the sender is gone.
3. Fix Windows/OSX filesystem incompatibilities (sender should trim filename just like receiver does)
4. Find a way to test the interwoven requests without interfering with go's race detector while reading the httptest Body.