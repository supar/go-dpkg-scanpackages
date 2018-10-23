package main

import (
	"bytes"
	"os"
	"testing"
)

var (
	controlDpkgScanResult = `Package: hello-world
Version: 1.0.0-1
Architecture: amd64
Maintainer: Pavel Rezunenko <paulrez@gmail.com>
Installed-Size: 7
Depends: php
Conflicts: python
Filename: fixtures/hello-world_1.0.0-1_amd64.deb
Size: 966
MD5sum: fa6dac2f36ed586b7ed5cebd9abe80d2
SHA1: 1da1a9bdf48b8bc7e7546b7c4162c997f04d60bc
SHA256: 11cd6d36cbde96622e8a57726c9002c9227f1a4347bd8eca77ac39ed816c989e
Description: This is test package
`
)

func Test_ReadFileFRomArchiveAndGzippedTar(t *testing.T) {
	var (
		arch *os.File
		err  error

		buf = bytes.NewBuffer(nil)
	)

	arch, err = os.Open("fixtures/hello-world_1.0.0-1_amd64.deb")
	if err != nil {
		t.Error(err)
	}

	err = controlFile(arch, buf)

	if !bytes.Contains(buf.Bytes(), []byte("Package: hello-world")) {
		t.Error("Expected Package information")
	}
}

func Test_ReadFileAndCountSums(t *testing.T) {
	var (
		arch *os.File
		err  error
		buf  []byte
	)

	arch, err = os.Open("fixtures/hello-world_1.0.0-1_amd64.deb")
	if err != nil {
		t.Error(err)
	}

	buf, err = fileSums(arch, SumMd5|SumSha1|SumSha256)
	if err != nil {
		t.Error(err)
	}

	if len(buf) == 0 {
		t.Error("Unexpected return: got 0")
	}
}

func Test_controlFile(t *testing.T) {
	filePath := "fixtures/hello-world_1.0.0-1_amd64.deb"

	res, err := os.Open(filePath)
	if err != nil {
		t.Error(err)
	}
	defer res.Close()

	var pkg []byte
	pkg, err = FileMetaData(res, SumMd5|SumSha1|SumSha256, "fixtures")
	if err != nil {
		t.Error(err)
	}

	if !bytes.Equal([]byte(controlDpkgScanResult), pkg) {
		t.Logf("%s", pkg)
		t.Logf("%d", len(pkg))
		t.Logf("%s", controlDpkgScanResult)
		t.Logf("%d", len([]byte(controlDpkgScanResult)))
		t.Error("not equal")
	}
}

func Benchmark_fileSums(b *testing.B) {
	filePath := "fixtures/hello-world_1.0.0-1_amd64.deb"

	res, err := os.Open(filePath)
	if err != nil {
		b.Error(err)
	}
	defer res.Close()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		res.Seek(0, 0)
		b.StartTimer()

		_, err = fileSums(res, SumMd5|SumSha1|SumSha256)
		if err != nil {
			b.Error(err)
		}
	}
}
