// Copyright 2014 Sven Taute. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package nbreader implements a non-blocking io.Reader.
//
// The blockSize defines the buffer size that is used to read from the underlying io.Reader.
//
// NBReader allows to specify two timeouts:
//
// Timeout: Read() returns after the specified timeout, even if no data has been read.
//
// ChunkTimeout: Read() returns if no data has been read for the specified time, even if the overall timeout has not been hit yet.
//
// ChunkTimeout must be smaller than Timeout.
//
// Example Usage:
//     // Create a NBReader that immediately returns on Read(), whether any data has been read or not
//     nbr := nbreader.NewNBReader(reader, 1 << 16)
//
//     // Create a NBReader that tries to return on Read() after no data has been read for 200ms
//     // or when the maximum timeout of 2 seconds is hit.
//     nbr := nbreader.NewNBReader(reader, 1 << 16, nbreader.Timeout(2000 * time.Millisecond), nbreader.ChunkTimeout(200 * time.Millisecond))
package nbreader

import (
	"bytes"
	"errors"
	"io"
	"time"
)

var errTimeout = errors.New("timeout")

// NBReader implements a non-blocking io.Reader.
type NBReader struct {
	blockSize    int
	reader       io.Reader
	dataChan     chan []byte
	buffer       bytes.Buffer
	chunkTimeout time.Duration
	forceTimeout time.Duration
	isEOF        bool
}

// Option implements options that can be passed to NewNBReader.
type Option func(r *NBReader)

// ChunkTimeout allows to set the timeout after which the end of a chunk of data is assumed and the read data is returned by Read().
func ChunkTimeout(duration time.Duration) Option {
	return func(r *NBReader) {
		r.chunkTimeout = duration
	}
}

// Timeout allows to set the timeout after which Read() returns, even if no data has been read.
func Timeout(duration time.Duration) Option {
	return func(r *NBReader) {
		r.forceTimeout = duration
	}
}

// NewNBReader returns a new NBReader with the given block size.
func NewNBReader(reader io.Reader, blockSize int, options ...Option) *NBReader {
	dataChan := make(chan []byte)
	r := NBReader{reader: reader, dataChan: dataChan, blockSize: blockSize}
	for _, option := range options {
		option(&r)
	}
	go r.readInput()
	return &r
}

// Read reads data into buffer. It returns the number of bytes read into buffer.
// At EOF, err will be io.EOF. Read() might still have read data when EOF is returned for the first time.
//
// Note: Read() is not safe for concurrent use.
func (r *NBReader) Read(buffer []byte) (int, error) {
	var (
		remaining   time.Duration
		nextTimeout time.Duration
		start       = time.Now()
		lastStart   = time.Now()
	)

	if len(buffer) <= r.buffer.Len() {
		ret, _ := r.buffer.Read(buffer)
		return ret, nil
	}

	if r.isEOF {
		return r.buffer.Read(buffer)
	}

	for r.buffer.Len() < len(buffer) {
		lastStart = time.Now()
		remaining = r.forceTimeout - time.Now().Sub(start)
		if r.chunkTimeout == 0 || r.chunkTimeout > remaining {
			nextTimeout = remaining
		} else {
			nextTimeout = r.chunkTimeout
		}
		_, err := r.readWithTimeout(r.buffer, nextTimeout)
		duration := time.Now().Sub(lastStart)
		if err == errTimeout {
			if duration >= r.chunkTimeout {
				break
			}
		}
		if err == io.EOF {
			r.isEOF = true
			break
		}
		if time.Now().Sub(start) >= r.forceTimeout {
			break
		}
	}
	ret, _ := r.buffer.Read(buffer)
	return ret, nil
}

// readInput is used by a goroutine to read data from the underlying io.Reader
func (r *NBReader) readInput() {
	for {
		tmp := make([]byte, r.blockSize)
		length, err := r.reader.Read(tmp)
		if err != nil {
			break
		}
		r.dataChan <- tmp[0:length]
	}
	close(r.dataChan)
}

// readWithTimeout consumes the data channel filled by readInput() and respects the set timeouts
func (r *NBReader) readWithTimeout(buffer bytes.Buffer, timeout time.Duration) (int, error) {
	select {
	case data, ok := <-r.dataChan:
		r.buffer.Write(data)
		if !ok {
			return len(data), io.EOF
		}
		return len(data), nil
	case <-time.After(timeout):
		return 0, errTimeout
	}
}
