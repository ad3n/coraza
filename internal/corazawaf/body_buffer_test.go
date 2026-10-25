// Copyright 2022 Juan Pablo Tosso and the OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package corazawaf

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/ad3n/coraza/v3/internal/environment"
	"github.com/ad3n/coraza/v3/types"
)

func TestBodyReaderMemory(t *testing.T) {
	br := NewBodyBuffer(types.BodyBufferOptions{
		TmpPath:     t.TempDir(),
		MemoryLimit: 500,
		Limit:       500,
	})
	if _, err := br.Write([]byte("test")); err != nil {
		t.Error(err)
	}
	buf := new(strings.Builder)
	reader, err := br.Reader()
	if err != nil {
		t.Error(err)
	}
	if _, err := io.Copy(buf, reader); err != nil {
		t.Error(err)
	}
	if buf.String() != "test" {
		t.Error("Failed to get BodyReader from memory")
	}
	_ = br.Reset()
}

func TestBodyReaderFile(t *testing.T) {
	if !environment.HasAccessToFS {
		return
	}

	for _, tt := range []struct {
		name          string
		alreadyClosed bool
	}{
		{name: "open file"},
		{name: "close error still removes file", alreadyClosed: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			br := NewBodyBuffer(types.BodyBufferOptions{TmpPath: t.TempDir(), MemoryLimit: 1, Limit: 100})
			if _, err := br.Write([]byte("test")); err != nil {
				t.Fatal(err)
			}

			reader, err := br.Reader()
			if err != nil {
				t.Fatal(err)
			}

			data, err := io.ReadAll(reader)
			if err != nil || string(data) != "test" {
				t.Fatalf("body = %q, %v; want test", data, err)
			}

			file := br.writer
			if _, err := os.Stat(file.Name()); err != nil {
				t.Fatal(err)
			}

			if tt.alreadyClosed {
				if err := file.Close(); err != nil {
					t.Fatal(err)
				}
			}

			err = br.Reset()
			if (err != nil) != tt.alreadyClosed {
				t.Fatalf("Reset error = %v, want error %t", err, tt.alreadyClosed)
			}

			if _, err := os.Stat(file.Name()); !os.IsNotExist(err) {
				t.Fatalf("temporary body file remains after Reset: %v", err)
			}
		})
	}
}

func TestBodyReaderWriteFromReader(t *testing.T) {
	waf := NewWAF()
	tests := []struct {
		name        string
		input       string
		limit       int64
		memoryLimit int64
		want        string
		wantErr     error
		readerError bool
		nilReader   bool
	}{
		{name: "memory", input: "test", limit: 4, memoryLimit: 5, want: "test"},
		{name: "short reader", input: "x", limit: 5, memoryLimit: 5, want: "x", wantErr: io.EOF},
		{name: "limited", input: "secret", limit: 2, memoryLimit: 5, want: "se"},
		{name: "empty", limit: 0, memoryLimit: 5},
		{name: "nil reader", limit: 4, memoryLimit: 5, nilReader: true, wantErr: errNilBodyReader},
		{name: "nil reader with zero limit", limit: 0, memoryLimit: 5, nilReader: true},
		{name: "negative", input: "test", limit: -1, memoryLimit: 5},
		{name: "file", input: "test", limit: 4, memoryLimit: 1, want: "test"},
		{name: "reader error", limit: 4, memoryLimit: 5, readerError: true, wantErr: io.ErrUnexpectedEOF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.memoryLimit < int64(len(tt.want)) && !environment.HasAccessToFS {
				return
			}

			br := NewBodyBuffer(types.BodyBufferOptions{
				TmpPath: t.TempDir(), MemoryLimit: tt.memoryLimit, Limit: 5,
			})
			defer func() {
				if err := br.Reset(); err != nil {
					t.Error(err)
				}
			}()

			src := strings.NewReader(tt.input)
			var input io.Reader = src
			if tt.readerError {
				input = iotest.ErrReader(io.ErrUnexpectedEOF)
			}

			if tt.nilReader {
				input = nil
			}

			n, err := waf.copyBodyN(br, input, tt.limit)
			if n != int64(len(tt.want)) || !errors.Is(err, tt.wantErr) {
				t.Fatalf("copy = %d, %v; want %d, %v", n, err, len(tt.want), tt.wantErr)
			}

			if src.Len() != len(tt.input)-len(tt.want) {
				t.Fatal("read beyond the requested limit")
			}

			reader, err := br.Reader()
			if err != nil {
				t.Fatal(err)
			}

			data, err := io.ReadAll(reader)
			if err != nil || string(data) != tt.want {
				t.Fatalf("body = %q, %v; want %q", data, err, tt.want)
			}
		})
	}
}

func TestWriteLimit(t *testing.T) {
	testCases := map[string]struct {
		initialBytes      []byte
		toBeWrittenBytes  []byte
		bodyBufferLimit   int64
		shouldReturnError bool
	}{
		"last byte written": {
			toBeWrittenBytes:  []byte("abc"),
			bodyBufferLimit:   3,
			shouldReturnError: false,
		},
		"over limit": {
			toBeWrittenBytes:  []byte("abc"),
			bodyBufferLimit:   2,
			shouldReturnError: true,
		},
		"over limit when limit already reached": {
			initialBytes:      []byte("abc"), // buffer will reach its limit
			toBeWrittenBytes:  []byte("a"),
			bodyBufferLimit:   3,
			shouldReturnError: true,
		},
	}
	for name, tCase := range testCases {
		t.Run(name, func(t *testing.T) {
			br := NewBodyBuffer(types.BodyBufferOptions{
				MemoryLimit: tCase.bodyBufferLimit,
				Limit:       tCase.bodyBufferLimit,
			})
			_, err := br.Write(tCase.initialBytes)
			if err != nil {
				t.Fatalf("unexpected error writing initial buffer: %s", err.Error())
			}
			writtenBytes, err := br.Write(tCase.toBeWrittenBytes)
			if tCase.shouldReturnError && err == nil {
				t.Fatal("expected error when writing above the limit")
			}
			if !tCase.shouldReturnError && writtenBytes != len(tCase.toBeWrittenBytes) {
				t.Fatalf("unexpected number of bytes written, want: %d, have: %d", len(tCase.toBeWrittenBytes), writtenBytes)
			}
			_ = br.Reset()
		})
	}
}

// See https://github.com/corazawaf/coraza-caddy/issues/48
func TestBodyBufferResetAndReadTheReader(t *testing.T) {
	br := NewBodyBuffer(types.BodyBufferOptions{
		MemoryLimit: 5,
		Limit:       5,
	})
	br.Write([]byte("test1")) // nolint

	r, _ := br.Reader()

	dest := make([]byte, 5)
	nRead, err := r.Read(dest)
	if err != nil {
		t.Fatalf("unexpected error when creating reader %s", err.Error())
	}
	if nRead != 5 {
		t.Fatalf("unexpected number of bytes read, want: %d, have: %d", 5, nRead)
	}

	err = br.Reset()
	if err != nil {
		t.Fatalf("unexpected error %s", err.Error())
	}

	nCopied, err := io.Copy(io.Discard, r)
	if err != nil {
		t.Fatalf("unexpected error %s", err.Error())
	}
	if nCopied != 0 {
		t.Fatalf("unexpected number of bytes read, want: %d, have: %d", 5, nCopied)
	}
}

func BenchmarkBodyCopy(b *testing.B) {
	for _, size := range []int{1024, 65536} {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			for _, pooled := range []bool{false, true} {
				b.Run(strconv.FormatBool(pooled), func(b *testing.B) {
					waf := NewWAF()
					body := NewBodyBuffer(types.BodyBufferOptions{MemoryLimit: 1 << 20, Limit: 1 << 20})
					data := bytes.Repeat([]byte("x"), size)
					reader := bytes.NewReader(data)
					copyBody := func() (int64, error) {
						return io.CopyN(body, reader, int64(size))
					}

					if pooled {
						copyBody = func() (int64, error) {
							return waf.copyBodyN(body, reader, int64(size))
						}
					}

					b.ReportAllocs()
					b.SetBytes(int64(size))

					for b.Loop() {
						reader.Reset(data)
						n, err := copyBody()
						if err != nil || n != int64(size) {
							b.Fatalf("copy = %d, %v", n, err)
						}

						if err := body.Reset(); err != nil {
							b.Fatal(err)
						}
					}
				})
			}

			b.Run("parallel", func(b *testing.B) {
				for _, pooled := range []bool{false, true} {
					b.Run(strconv.FormatBool(pooled), func(b *testing.B) {
						waf := NewWAF()
						data := bytes.Repeat([]byte("x"), size)
						b.ReportAllocs()
						b.SetBytes(int64(size))
						b.RunParallel(func(pb *testing.PB) {
							body := NewBodyBuffer(types.BodyBufferOptions{MemoryLimit: 1 << 20, Limit: 1 << 20})
							reader := bytes.NewReader(data)
							for pb.Next() {
								reader.Reset(data)
								copyBody := func() (int64, error) {
									return io.CopyN(body, reader, int64(size))
								}

								if pooled {
									copyBody = func() (int64, error) {
										return waf.copyBodyN(body, reader, int64(size))
									}
								}

								n, err := copyBody()
								if err != nil || n != int64(size) || !bytes.Equal(body.buffer.Bytes(), data) {
									b.Fatalf("concurrent body copy = %d, %v", n, err)
								}

								if err := body.Reset(); err != nil {
									b.Fatal(err)
								}
							}
						})
					})
				}
			})
		})
	}
}
