package main

import (
	"hash/fnv"
	"path/filepath"
	"os"
	"io"
	baseenc "encoding/base32"
	"encoding/gob"
)

type Persisted struct {
	Exists bool
	PersistedHash FileHash
}

func (dc * datacrypt) GetPersistId(file FileData) (FileId, error){
	id := file.getFileId(dc)
	var thing Persisted 
	err := dc.Files.Get(id.ID[:16], &thing)
	return id,err
}

func (dc * datacrypt) GenPersistId(file FileData) FileId {
	id := file.getFileId(dc)
	var thing Persisted
	thing.Exists = false
	err := dc.Files.Put(id.ID[:16], thing)
	if err != nil {
		panic(err)
	}
	return id
}

func (dc * datacrypt) FilePersisted(id FileId) (FileHash, error){
	var thing Persisted 
	err := dc.Files.Get(id.ID[:16], &thing)
	return thing.PersistedHash,err	
}

func (dc * datacrypt) LookupPersisted(hash FileHash) error{
	err := dc.FileHashes.Get(hash.ToBytes(), nil)
	return err
}

func (dc * datacrypt) MarkPersisted(hash FileHash){
	err := dc.FileHashes.Put(hash.ToBytes(), nil)
	if err != nil {
		panic(err)
	}
}

func (dc * datacrypt) PersistData(file FileData) (FileHash, error){
	if file.IsDirectory {
		panic("Directories cannot be persisted")
	}
	readFile := filepath.Join(dc.dataFolder, file.Folder, file.Name);
	readfilestr, err := os.Open(readFile)
	if err != nil {
		return FileHash{}, err
	}
	fileid, err := dc.GetPersistId(file)
	if err != nil {
		panic(err)
	}
	
	hasher := fnv.New128()
	if _, err := io.Copy(hasher, readfilestr); err != nil {
		panic(err)
	}
	
	hsh := hasher.Sum(nil)

	var fhash [16]byte

	copy(fhash[:], hsh)
	
	chash,err := dc.FilePersisted(fileid)
	newhash := FileHash { Hash: fhash, Size: file.Size}
	if newhash == chash {
		return chash,nil
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

	
	p := Persisted {Exists: true, PersistedHash: newhash}
	dc.Files.Put(fileid.ID[:16], p)
	dc.MarkPersisted(p.PersistedHash)
	return p.PersistedHash, nil
}

func (dc * datacrypt) SetFileLet(fileid FileId, let FileLet) error{
	return dc.Lets.Put(fileid.ID[:16], let)
}

func (dc * datacrypt) GetFileLet(fileid FileId) (FileLet,error) {
	var f FileLet
	err := dc.Lets.Get(fileid.ID[:16], &f)
	return f,err
}

func (dc * datacrypt) GetChangeHash(f FileId) ChangeHash {
	var ch ChangeHash
	err := dc.Change.Get(f.ID[:], &ch)
	if err != nil{
		return ChangeHash{}
	}
	return ch
}

func (dc * datacrypt) FileDeleted(f FileId) bool {
	var p Persisted
	err := dc.Files.Get(f.ID[:], &p)
	if err != nil {
		return false
	}
	return p.Exists == false
}

func (dc * datacrypt) FileExists(file FileData) bool{
	filepath := filepath.Join(dc.dataFolder, file.Folder, file.Name);
	return fileExists(filepath)
}

func (dc * datacrypt) PushCommit(change ChangeData){
	newhash := change.Hash()
	err := dc.Change.Put(change.ID.ID[:], newhash);
	if err != nil {
		panic(err)
	}
	outFile := filepath.Join(dc.commitFolder, baseenc.StdEncoding.EncodeToString(change.ID.ID[:]))	
	f,err := os.OpenFile(outFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		panic(err)
	}
	enc := gob.NewEncoder(f)
	enc.Encode(change)
	f.Close()
}
