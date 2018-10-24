package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/blakesmith/ar"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	SumMd5 = 1 << iota
	SumSha1
	SumSha256
)

type fileCallback func(io.Reader) error

var (
	// ErrPackageFieldRequired reports if control does not contains Package field
	ErrPackageFieldRequired = errors.New("no Package field in control file")
	// ErrPackageFieldMultiple reports if control file contains multiple Package fields.
	// It means that deb archive is group of packages
	ErrPackageFieldMultiple = errors.New("multiple Package field")
)

// FileMetaData returns axtracted control file from debian archive with additional fields:
// Size, Filename and controls sums
func FileMetaData(file *os.File, sumMask uint8, filePrefix string) (b []byte, err error) {
	var (
		buf *bytes.Buffer
	)

	buf = bytes.NewBuffer(nil)

	if err = controlFile(file, buf); err != nil {
		return
	}

	if l := bytes.Count(buf.Bytes(), []byte("Package")); l == 0 {
		return nil, ErrPackageFieldRequired
	} else if l > 1 {
		return nil, ErrPackageFieldMultiple
	}

	file.Seek(0, 0)

	// Order search: Section, Priority, Description
	breakPoint := [][]byte{
		[]byte("Section"),
		[]byte("Priority"),
		[]byte("Description"),
	}

	for i := range breakPoint {
		point := bytes.LastIndex(buf.Bytes(), breakPoint[i])

		if point > -1 {
			var (
				info os.FileInfo
				sums []byte
			)

			right := make([]byte, len(buf.Bytes()[point:]))
			copy(right, buf.Bytes()[point:])
			buf.Truncate(point)

			// write file size
			if info, err = file.Stat(); err != nil {
				return
			}
			buf.Write([]byte("Filename: " + filepath.Join(filePrefix, info.Name()) + "\n"))
			buf.Write([]byte("Size: " + strconv.FormatInt(info.Size(), 10) + "\n"))

			sums, err = fileSums(file, sumMask)
			if err != nil {
				return
			}
			buf.Write(sums)

			buf.Write(right)

			break
		}
	}

	return buf.Bytes(), err
}

// controlFile copies control file data to the buffer
func controlFile(file *os.File, buf io.Writer) (err error) {
	var (
		cb fileCallback
	)

	// File ready let's read
	cb = func(r io.Reader) error {
		return tarFile(r, "control", func(r io.Reader) error {
			_, err := io.Copy(buf, r)

			return err
		})
	}

	if err = arFile(file, "control.tar.gz", cb); err != nil {
		return
	}

	return
}

// fileSums returns MD5, SHA1, SHA256 checksums of the data
func fileSums(file *os.File, mask uint8) (b []byte, err error) {
	var (
		hh = make([]hash.Hash, 0, 3)
	)

	// create md5 writer
	if mask&SumMd5 != 0 {
		hh = append(hh, md5.New())
	}

	// create sha1 writer
	if mask&SumSha1 != 0 {
		hh = append(hh, sha1.New())
	}

	// create sha256 writer
	if mask&SumSha256 != 0 {
		hh = append(hh, sha256.New())
	}

	if len(hh) == 0 {
		// nothing to do cause we checked all we support
		return
	}

	// To resolve casting from hash.Hash to io.Writer
	ws := make([]io.Writer, len(hh))
	for i := range hh {
		ws[i] = hh[i]
	}

	// Create multiwriter and read file
	wr := io.MultiWriter(ws...)
	if _, err = io.Copy(wr, file); err != nil {
		return
	}

	// Read sums
	b = make([]byte, 0, 256)
	for i := len(hh) - 1; i >= 0; i-- {
		hexLen := hex.EncodedLen(hh[i].Size())

		// grow slice
		b = unshift(b, []byte("\n"))
		b = expandLeft(b, hexLen)
		hex.Encode(b, hh[i].Sum(nil))

		switch hexLen {
		case 32:
			// MD5
			b = unshift(b, []byte("MD5sum: "))
		case 40:
			// SHA1
			b = unshift(b, []byte("SHA1: "))
		case 64:
			// SHA256
			b = unshift(b, []byte("SHA256: "))
		}

	}

	return
}

// expandLeft expands slice to left
func expandLeft(slice []byte, shift int) []byte {
	l := len(slice)

	// Capacity safe
	// Slice is full; must grow.
	// We double its size and add 1, so if the size is zero we still grow.
	if cap(slice) <= l+shift {
		newSlice := make([]byte, l, 2*l+shift+1)
		copy(newSlice, slice)
		slice = newSlice
	}

	slice = slice[:l+shift]

	if l > 0 {
		copy(slice[shift:], slice[:l])
	}

	return slice
}

// unshift inserts data to the begining
func unshift(buf, data []byte) []byte {
	l := len(data)

	buf = expandLeft(buf, l)

	copy(buf[:l], data)

	return buf
}

// arFile reads file data from the debian archive
func arFile(file io.Reader, sFile string, cb fileCallback) (err error) {
	var (
		arReader *ar.Reader
		header   *ar.Header
	)

	arReader = ar.NewReader(file)

	for {
		if header, err = arReader.Next(); err != nil {
			// Reached end don't treat error
			if err == io.EOF {
				err = nil
			}
			return
		}

		if strings.HasPrefix(header.Name, sFile) {
			if strings.HasSuffix(sFile, ".gz") {
				var gz *gzip.Reader

				if gz, err = gzip.NewReader(arReader); err != nil {
					return
				}

				return cb(gz)
			}

			return cb(arReader)
		}
	}

	return
}

// tarFile reads file from tar archive
func tarFile(file io.Reader, sFile string, cb fileCallback) (err error) {
	var (
		header    *tar.Header
		tarReader *tar.Reader
	)

	tarReader = tar.NewReader(file)

	for {
		if header, err = tarReader.Next(); err != nil {
			// Reached end don't treat error
			if err == io.EOF {
				err = nil
			}
			return
		}

		if strings.HasSuffix(header.Name, sFile) {
			return cb(tarReader)
		}
	}

	return
}
