package bitcask

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const (
	recordHeaderSize = 1 + 8 + 8
	dataFilePrefix   = "data-"
	dataFileSuffix   = ".db"
)

const maxFileSize = 128 << 20 // 128 MB

var (
	ErrKeyNotFound = errors.New("key not found")
)

type entry struct {
	fileId int64
	offset int64
	size   int64
}

type Bitcask struct {
	dir        string
	mu         sync.RWMutex
	keydir     map[string]entry
	files      map[int64]*os.File
	currID     int64
	currFile   *os.File
	currOffset int64
	bufw       *bufio.Writer
}

func extractFileId(path string) int64 {
	base := filepath.Base(path)
	base = strings.TrimPrefix(base, dataFilePrefix)
	base = strings.TrimSuffix(base, dataFileSuffix)
	id, _ := strconv.ParseInt(base, 10, 64)
	return id
}

func (bc *Bitcask) Keys() []string {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	keys := make([]string, 0, len(bc.keydir))
	for k := range bc.keydir {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (bc *Bitcask) Close() error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	if bc.bufw != nil {
		if err := bc.bufw.Flush(); err != nil {
			return err
		}
	}
	for _, f := range bc.files {
		if f != nil {
			f.Sync()
			f.Close()
		}
	}
	return nil
}

func (bc *Bitcask) ScanFile(fid int64, file *os.File) error {
	var off int64 = 0
	r := bufio.NewReader(file)
	for {
		hdr := make([]byte, recordHeaderSize)
		n, err := io.ReadFull(r, hdr)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			return err
		}

		offIncr := int64(n)
		flags := hdr[0]
		keyLen := int64(binary.BigEndian.Uint64(hdr[1:9]))
		valLen := int64(binary.BigEndian.Uint64(hdr[9:17]))

		key := make([]byte, keyLen)
		if _, err := io.ReadFull(r, key); err != nil {
			return err
		}
		offIncr += keyLen
		if valLen > 0 {
			if _, err := io.CopyN(io.Discard, r, valLen); err != nil {
				return err
			}
			offIncr += valLen
		}
		k := string(key)
		ent := entry{
			fileId: fid,
			offset: off,
			size:   offIncr,
		}
		if flags&0x1 == 0x1 {
			delete(bc.keydir, k)
		} else {
			bc.keydir[k] = ent
		}

		off += offIncr
	}
	return nil
}

func (bc *Bitcask) Delete(key string) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	rec := make([]byte, recordHeaderSize+len(key))
	rec[0] = 0x1
	binary.BigEndian.PutUint64(rec[1:9], uint64(len(key)))
	binary.BigEndian.PutUint64(rec[9:17], 0)
	copy(rec[17:], []byte(key))

	if _, err := bc.bufw.Write(rec); err != nil {
		return err
	}
	if err := bc.bufw.Flush(); err != nil {
		return err
	}
	delete(bc.keydir, key)
	bc.currOffset += int64(len(rec))
	return nil
}

func (bc *Bitcask) Get(key string) ([]byte, error) {
	bc.mu.RLock()
	ent, ok := bc.keydir[key]
	bc.mu.RUnlock()
	if !ok {
		return nil, ErrKeyNotFound
	}
	bc.mu.RLock()
	file, ok := bc.files[ent.fileId]
	bc.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("data file %d not found", ent.fileId)
	}
	buf := make([]byte, ent.size)
	if _, err := file.ReadAt(buf, ent.offset); err != nil {
		return nil, err
	}
	if len(buf) < recordHeaderSize {
		return nil, fmt.Errorf("corrupt record")
	}
	flags := buf[0]
	keyLen := int64(binary.BigEndian.Uint64(buf[1:9]))
	valLen := int64(binary.BigEndian.Uint64(buf[9:17]))
	if flags&0x1 == 0x1 {
		return nil, ErrKeyNotFound
	}
	start := recordHeaderSize + keyLen
	if int64(len(buf)) < start+valLen {
		return nil, fmt.Errorf("corrupt value length")
	}
	value := make([]byte, valLen)
	copy(value, buf[start:start+valLen])
	return value, nil
}

func (bc *Bitcask) Put(key string, value []byte) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if err := bc.RotateFile(); err != nil {
		return err
	}

	rec := make([]byte, recordHeaderSize+len(key)+len(value))
	rec[0] = 0x0
	binary.BigEndian.PutUint64(rec[1:9], uint64(len(key)))
	binary.BigEndian.PutUint64(rec[9:17], uint64(len(value)))
	copy(rec[17:17+len(key)], []byte(key))
	copy(rec[17+len(key):], value)
	if _, err := bc.bufw.Write(rec); err != nil {
		return err
	}
	if err := bc.bufw.Flush(); err != nil {
		return err
	}
	ent := entry{fileId: bc.currID, offset: bc.currOffset, size: int64(len(rec))}
	bc.keydir[key] = ent
	bc.currOffset += int64(len(rec))
	return nil
}

func Open(dir string) (*Bitcask, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	bc := &Bitcask{
		dir:    dir,
		keydir: make(map[string]entry),
		files:  make(map[int64]*os.File),
	}

	files, err := filepath.Glob(filepath.Join(dir, dataFilePrefix+"*"+dataFileSuffix))
	if err != nil {
		return nil, err
	}

	sort.Slice(files, func(i, j int) bool {
		iid := extractFileId(files[i])
		jid := extractFileId(files[j])
		return iid < jid
	})

	var maxId int64 = -1
	for _, fpath := range files {
		fid := extractFileId(fpath)
		if fid > maxId {
			maxId = fid
		}
		file, err := os.OpenFile(fpath, os.O_RDWR, 0o644)
		if err != nil {
			return nil, err
		}
		bc.files[fid] = file
		if err := bc.ScanFile(fid, file); err != nil {
			file.Close()
			return nil, fmt.Errorf("scan file %s: %w", fpath, err)
		}
		fmt.Println("Bitcask : ", bc)
		fmt.Println("")
	}
	if maxId == -1 {
		bc.currID = 0
		currPath := filepath.Join(dir, dataFilePrefix+"0"+dataFileSuffix)
		file, err := os.OpenFile(currPath, os.O_CREATE|os.O_RDWR, 0o644)
		if err != nil {
			return nil, err
		}
		bc.files[0] = file
		bc.currFile = file
		bc.currOffset = 0
		bc.bufw = bufio.NewWriterSize(file, 4096)
	} else {
		bc.currID = maxId
		file := bc.files[maxId]
		off, err := file.Seek(0, io.SeekEnd)
		if err != nil {
			file.Close()
			return nil, err
		}
		bc.currFile = file
		bc.currOffset = off
		bc.bufw = bufio.NewWriterSize(file, 4096)
	}

	return bc, nil
}

func (bc *Bitcask) RotateFile() error {
	if bc.currOffset < maxFileSize {
		return nil
	}

	if bc.bufw != nil {
		if err := bc.bufw.Flush(); err != nil {
			return err
		}
	}
	bc.currFile.Sync()
	bc.currFile.Close()

	bc.currID++
	path := filepath.Join(bc.dir, dataFilePrefix+strconv.FormatInt(bc.currID, 10)+dataFileSuffix)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	bc.files[bc.currID] = file
	bc.currFile = file
	bc.currOffset = 0
	bc.bufw = bufio.NewWriterSize(file, 4096)
	return nil
}

func (bc *Bitcask) Entries() (map[string][]byte, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	result := make(map[string][]byte, len(bc.keydir))
	for k := range bc.keydir {
		val, err := bc.Get(k)
		if err != nil {
			return nil, err
		}
		result[k] = val
	}
	return result, nil
}

func (bc *Bitcask) RestoreFromSnapshot(data map[string][]byte) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	for _, f := range bc.files {
		f.Close()
	}
	bc.files = make(map[int64]*os.File)
	bc.keydir = make(map[string]entry)

	bc.currID = 0
	path := filepath.Join(bc.dir, dataFilePrefix+"0"+dataFileSuffix)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	bc.files[0] = file
	bc.currFile = file
	bc.currOffset = 0
	bc.bufw = bufio.NewWriterSize(file, 4096)

	for k, v := range data {
		if err := bc.Put(k, v); err != nil {
			return err
		}
	}
	return nil
}

func (bc *Bitcask) ApplyCommand(op, key string, val []byte) error {
	switch op {
	case "PUT":
		return bc.Put(key, val)
	case "DEL":
		return bc.Delete(key)
	default:
		return fmt.Errorf("unknown operation: %s", op)
	}
}
