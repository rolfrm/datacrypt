package main

import (
	"io/ioutil"
	"hash/fnv"
	"path/filepath"
	"io"
	"os"
	"fmt"
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
	tmp,err := ioutil.TempFile(dc.localFolder, "TMP_")
	if err != nil {
		return FileHash{}, err
	}
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
	readfilestr.Seek(0, 0)
	
	writer := CompressionWriter(tmp, dc.key)

	
	if _, err := io.Copy(writer, readfilestr); err != nil {
		panic(err)
	}
	writer.Close()

	fmt.Println(hsh)
	return FileHash{}, nil
	
	
}
