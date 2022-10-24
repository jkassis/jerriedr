package core

import (
	"container/ring"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// FileRing vends writers attached to files with names that include a serial suffix
type FileRing struct {
	BasePath       string
	KeepNum        int
	baseDir        string
	baseName       string
	indexCurrent   int
	writeIndexRing *ring.Ring
	readIndexRing  *ring.Ring
}

// Init scans the target directory to sync the internal ring buffer of file indexes with
// those found on the file system
func (fr *FileRing) Init() error {
	fr.writeIndexRing = ring.New(fr.KeepNum)
	fr.readIndexRing = fr.writeIndexRing

	fr.baseName = filepath.Base(fr.BasePath)
	fr.baseDir = filepath.Dir(fr.BasePath)

	files, err := ioutil.ReadDir(fr.baseDir)
	if err != nil {
		return err
	}

	var indexes []int
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if !strings.HasPrefix(file.Name(), fr.baseName) {
			continue
		}
		suffix := file.Name()[len(fr.baseName):]
		if !strings.HasPrefix(suffix, "-") {
			continue
		}

		digits := suffix[1:]
		index, err := strconv.ParseInt(digits, 0, 0)
		if err != nil {
			continue
		}

		indexes = append(indexes, int(index))
	}

	// sort and add to the ring buffer
	sort.Ints(indexes)
	for _, index := range indexes {
		fr.indexCurrent = index
		fr.writeIndexRing = fr.writeIndexRing.Next()
		fr.writeIndexRingPut(fr.indexCurrent)
	}
	return nil
}

func (fr *FileRing) indexToFilename(index int) string {
	return fmt.Sprintf("%s-%9d", fr.baseDir, index)
}

// writeIndexRingPut adds an index to the next ring of the buffer.
// If an index is already found there, this tries to delete it.
func (fr *FileRing) writeIndexRingPut(index int) {
	oldValue := fr.writeIndexRing.Value
	if oldValue != nil {
		oldIndex := oldValue.(int)
		os.Remove(fr.indexToFilename(oldIndex))
	}
	fr.writeIndexRing.Value = index
}

// WriteNext advances the write index... the index appended to the name of the file this generates
// it then reads the data at the next position in the internal ring buffer
// if it finds an index at that position in the ring, it attempts to delete the associated file
// but ignores errors with deletion
//
// it then saves the new index in the ring, creates a new file associated with that index, and
// returns a io.Writer to the file
//
// This does create a small bug... if the ring is reloaded at a moment
// when the indexes of the file ring span max int and 0, the ring will load out of order
func (fr *FileRing) WriteNext() (io.Writer, error) {
	fr.indexCurrent = fr.indexCurrent + 1
	fr.writeIndexRing = fr.writeIndexRing.Next()
	fr.writeIndexRingPut(fr.indexCurrent)
	return os.Create(fr.indexToFilename(fr.indexCurrent))
}

// ReadReset sets the readIndexRing to the current writeIndexRing
// Calling ReadNext will get a reader for the *oldest* file in the ring.
// Calling ReadPrev will get a reader for the *newest* file in the ring.
func (fr *FileRing) ReadReset() {
	fr.readIndexRing = fr.writeIndexRing
}

// ReadCurrent returns a reader to the file associated with the index stored at the
// current position of the read ring buffer. Clients should be careful when calling
// this right after ReadReset since the associated file may be open for writing.
// If there is no file index stored at the current position, this returns nil, nil
func (fr *FileRing) ReadCurrent() (io.Reader, error) {
	index := fr.readIndexRing.Value
	if index == nil {
		return nil, nil
	}
	return os.Open(fr.indexToFilename(index.(int)))
}

// ReadNext moves the readIndexRing fwd until it finds a ring element with a valid index.
// It then returns a reader for the file that matches that index
// It returns nil, nil when it loops around to the current write index. In this case,
// the client can call ReadCurrent if it knows that it is not currently writing to the
// backing file.
func (fr *FileRing) ReadNext() (io.Reader, error) {
	for {
		fr.readIndexRing = fr.readIndexRing.Next()
		if fr.readIndexRing == fr.writeIndexRing {
			// we went all the way around
			return nil, nil
		}

		index := fr.readIndexRing.Value
		if index == nil {
			continue
		}
		return os.Open(fr.indexToFilename(index.(int)))
	}
}

// ReadPrev moves the readIndexRing bwd until it finds a ring element with a valid index.
// It then returns a reader for the file that matches that index
// It returns nil, nil when it loops around to the current write index. In this case,
// the client can call ReadCurrent if it knows that it is not currently writing to the
// backing file.
func (fr *FileRing) ReadPrev() (io.Reader, error) {
	for {
		fr.readIndexRing = fr.readIndexRing.Prev()
		if fr.readIndexRing == fr.writeIndexRing {
			// we went all the way around
			return nil, nil
		}

		index := fr.readIndexRing.Value
		if index == nil {
			continue
		}
		return os.Open(fr.indexToFilename(index.(int)))
	}
}
