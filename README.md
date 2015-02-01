go-nbreader: a non-blocking io.Reader for go
============================================

go-nbreader provides a non-blocking io.Reader for go (golang).

NBReader allows to specify two timeouts:

Timeout: Read() returns after the specified timeout, even if no data has been read.

ChunkTimeout: Read() returns if no data has been read for the specified time, even if the overall timeout has not been hit yet.

ChunkTimeout must be smaller than Timeout.

When the internal buffer contains at least blockSize bytes, Read() returns regardless of the specified timeouts.

Example Usage:

	// Create a NBReader that immediately returns on Read(), whether any data has been read or not
	nbr := nbreader.NewNBReader(reader, 1 << 16)

	// Create a NBReader that tries to return on Read() after no data has been read for 200ms
	// or when at least 64k bytes have been read or the maximum timeout of 2 seconds is hit.
	nbr := nbreader.NewNBReader(reader, 1 << 16, nbreader.Timeout(2000 * time.Millisecond), nbreader.ChunkTimeout(200 * time.Millisecond))

The full documentation can be found here: <http://godoc.org/github.com/svent/go-nbreader>
