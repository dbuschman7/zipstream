package zipstream

import (
	"archive/zip"
	"bytes"
	"io"
	"log"
	"net/http"
	"os"
	"testing"

	hu "github.com/dustin/go-humanize"

	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
)

var file22 = "https://github.com/golang/go/archive/refs/tags/go1.22.1.zip"
var file16 = "https://github.com/golang/go/archive/refs/tags/go1.16.10.zip"

func TestStreamReader(t *testing.T) {
	resp, err := http.Get(file22)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	zr := NewReader(resp.Body)

	var totalFileCount int64
	var totalFileSize uint64
	var compressedSize uint64

	for {
		e, err := zr.GetNextEntry()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("unable to get next entry: %s", err)
		}

		hash256 := sha256.New()
		hash1 := sha1.New()
		hashmd5 := md5.New()

		if !e.IsDir() {
			rc, err := e.Open()
			if err != nil {
				log.Fatalf("unable to open zip file: %s", err)
			}
			content, err := io.ReadAll(rc)
			if err != nil {
				log.Fatalf("read zip file content fail: %s", err)
			}

			log.Println("file length:", len(content))

			totalFileCount++
			totalFileSize += uint64(len(content))
			compressedSize += e.CompressedSize64

			hash256.Write(content)
			hash1.Write(content)
			hashmd5.Write(content)

			if uint64(len(content)) != e.UncompressedSize64 {
				log.Fatalf("read zip file length not equal with UncompressedSize64")
			}
			if err := rc.Close(); err != nil {
				log.Fatalf("close zip entry reader fail: %s", err)
			}

			log.Printf("entry name: %s, modify time: %v, compressed size: %d, uncompressed size: %d, crc32: %d, method: %d, flags: %d, extra: %s sha256: %x, sha1: %x, md5: %x\n",
				e.Name, e.Modified.UTC().UnixMilli(), e.CompressedSize64, e.UncompressedSize64, e.CRC32, e.Method, e.Flags, e.Extra, hash256.Sum(nil), hash1.Sum(nil), hashmd5.Sum(nil))
		}

	}

	log.Printf("total file count: %s, compressed: %s uncompressed: %s\n", hu.Comma(totalFileCount), hu.Bytes(compressedSize), hu.Bytes(totalFileSize))
}

func TestNewReader(t *testing.T) {

	f, err := os.Open("testdata/example.zip")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	zipFile, err := os.ReadFile("testdata/example.zip")
	if err != nil {
		t.Fatal(err)
	}

	az, err := zip.NewReader(f, int64(len(zipFile)))
	if err != nil {
		t.Fatal(err)
	}

	fileMap := make(map[string]*zip.File, len(az.File))

	for _, zf := range az.File {
		fileMap[zf.Name] = zf
	}

	z := NewReader(f)

	for {
		entry, err := z.GetNextEntry()
		if err == io.EOF {
			// iterator over
			break
		}

		zf, ok := fileMap[entry.Name]
		if !ok {
			t.Fatalf("not expected file: %s", entry.Name)
		}
		delete(fileMap, entry.Name)

		if entry.Comment != zf.Comment ||
			entry.ReaderVersion != zf.ReaderVersion ||
			entry.IsDir() != zf.Mode().IsDir() ||
			entry.Flags != zf.Flags ||
			entry.Method != zf.Method ||
			!entry.Modified.Equal(zf.Modified) ||
			entry.CRC32 != zf.CRC32 ||
			//bytes.Compare(entry.Extra, zf.Extra) != 0 || // local file header's extra data may not same as central directory header's extra data
			entry.CompressedSize64 != zf.CompressedSize64 ||
			entry.UncompressedSize64 != zf.UncompressedSize64 {
			t.Fatal("some local file header attr is incorrect")
		}

		if !entry.IsDir() {
			rc, err := entry.Open()
			if err != nil {
				t.Fatalf("open zip file entry err: %s", err)
			}

			entryFileContents, err := io.ReadAll(rc)
			if err != nil {
				t.Fatalf("read entry file contents fail: %s", err)
			}

			ziprc, err := zf.Open()
			if err != nil {
				t.Fatal(err)
			}
			zipFileContents, err := io.ReadAll(ziprc)
			if err != nil {
				t.Fatal(err)
			}

			if bytes.Compare(entryFileContents, zipFileContents) != 0 {
				t.Fatal("the zip entry file contents is incorrect")
			}

			if err := rc.Close(); err != nil {
				t.Fatalf("close zip file entry reader err: %s", err)
			}
			_ = ziprc.Close()
		}
	}

	if len(fileMap) != 0 {
		t.Fatal("the resolved entry count is incorrect")
	}

}
