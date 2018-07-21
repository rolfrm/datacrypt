package main

import "time"
import "encoding/gob"
import "encoding/binary"
import "bytes"
import "hash/fnv"
import "fmt"

type FileId struct {
	ID [16]byte
}

type FileHash struct {
	Size int64
	Hash [16]byte
}

func (f FileHash) ToBytes() []byte {
	var buf bytes.Buffer
	buf.Write(f.Hash[:])
	var sizeBytes [8]byte
	cnt := binary.PutVarint(sizeBytes[:], f.Size)
	buf.Write(sizeBytes[:cnt])
	return buf.Bytes()
}

type ChangeHash struct {
	Hash [16]byte
}

type FilePersistence interface {
	GetPersistId(file FileData) (FileId, error)
	GenPersistId(file FileData) FileId
	FilePersisted(fid FileId) (FileHash, error)
	PersistData(file FileData) (FileHash, error)
	GetFileLet(file FileId) (FileLet, error)
	SetFileLet(file FileId, let FileLet) error
	GetChangeHash(fid FileId) ChangeHash
	FileDeleted(fid FileId) bool
	FileExists(file FileData) bool
	PushCommit(change ChangeData)
}

type Change interface {
}

func (data *ChangeData) Hash() ChangeHash {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	enc.Encode(data)
	hsh := fnv.New128()
	sum := hsh.Sum(buf.Bytes())

	var ch ChangeHash
	for i := 0; i < 16; i++ {
		ch.Hash[i] = sum[i]
	}
	return ch
}

type ChangeData struct {
	ID     FileId
	Parent ChangeHash
	Data   Change
}

type ItemCreated struct {
	Folder      string
	Name        string
	IsDirectory bool
}

type FileDataChanged struct {
	Size    int64
	ModTime time.Time
}

type PersistedFileData struct {
	Hash FileHash
}

type FileDeleted struct {
}

func getFileUpdates(dc FilePersistence, file FileData) {

	id, err := dc.GetPersistId(file)
	if err != nil {
		id = dc.GenPersistId(file)
		var cd ChangeData
		cd.ID = id
		var ic ItemCreated
		cd.Data = ic
		ic.Name = file.Name
		ic.Folder = file.Folder
		ic.IsDirectory = file.IsDirectory
		dc.PushCommit(cd)
	}

	if dc.FileExists(file) == false && dc.FileDeleted(id) == false {
		var cd ChangeData
		cd.ID = id
		cd.Parent = dc.GetChangeHash(id)
		cd.Data = FileDeleted{}
		dc.PushCommit(cd)
		return
	}

	flet, err := dc.GetFileLet(id)

	if err == nil && flet.Size == file.Size && flet.ModTime == file.ModTime {
		return
	}

	if file.IsDirectory {
		return
	}
	nhsh, err := dc.PersistData(file)
	if err != nil {
		panic(fmt.Sprintf("What now!?! %v", err))
	}
	{
		var cd ChangeData
		cd.ID = id
		cd.Parent = dc.GetChangeHash(id)
		var fd PersistedFileData
		fd.Hash = nhsh
		dc.PushCommit(cd)
	}

	{
		flet := file.ToFileLet()
		var cd ChangeData
		cd.ID = id
		cd.Parent = dc.GetChangeHash(id)
		var fd FileDataChanged
		fd.Size = flet.Size
		fd.ModTime = flet.ModTime
		dc.PushCommit(cd)
	}
	dc.SetFileLet(id, file.ToFileLet())
}
