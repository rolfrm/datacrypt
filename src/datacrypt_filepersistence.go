package main

import (
	"hash/fnv"
	"path/filepath"
	"os"
	"io"
	baseenc "encoding/base32"
)

type Persisted struct {
	Exists bool
	PersistedHash FileHash
}

func (dc * datacrypt) GetPersistId(file FileData) (FileId, error){
	id := file.getFileId(dc)
	var thing Persisted 
	err := dc.dbGet([]byte("files"), id.id[:16], &thing)
	return id,err
}

func (dc * datacrypt) GenPersistId(file FileData) FileId {
	id := file.getFileId(dc)
	var thing Persisted
	thing.Exists = false
	err := dc.dbPut([]byte("files"), id.id[:16], thing)
	if err != nil {
		panic(err)
	}
	return id
}

func (dc * datacrypt) FilePersisted(id FileId) (FileHash, error){
	var thing Persisted 
	err := dc.dbGet([]byte("files"), id.id[:16], &thing)
	return thing.PersistedHash,err	
}

func (dc * datacrypt) PersistData(file FileData) (FileHash, error){
	
	readFile := filepath.Join(dc.dataFolder, file.Folder, file.Name);
	readfilestr, err := os.Open(readFile)
	if err != nil {
		return FileHash{}, err
	}

	hasher := fnv.New128()
	if _, err := io.Copy(hasher, readfilestr); err != nil {
		panic(err)
	}

	hsh := hasher.Sum(nil)
	var hash [16]byte
	copy(hash[:], hsh[:16])
	var existing []byte
	err = dc.dbGet([]byte("files"), hash[:16], &existing)
	if err == nil {
		return FileHash{ id: hash }, nil
	}
	
	readfilestr.Seek(0, 0)

	outFile := filepath.Join(dc.commitFolder, baseenc.StdEncoding.EncodeToString(hsh))
	f, err := os.OpenFile(outFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	
	if err != nil {
		return FileHash{}, err
	}
	
	writer := CompressionWriter(f, dc.key)

	written, err := io.Copy(writer, readfilestr)
	if err != nil {
		panic(err)
	}
	writer.Close()
	f.Close()
	os.Truncate(outFile, written)
	return FileHash{ id: hash }, nil
	
	
}
