package main

import "time"
import "encoding/gob"
import "bytes"
import "hash/fnv"

type FileId struct {
	id [16]byte
}

type FileHash struct {
	id [16]byte
}

type ChangeHash struct{
	id [16]byte
}

type FilePersistence interface {

	GetPersistId(file FileData) (FileId, error)
	GenPersistId(file FileData) FileId
	FilePersisted(fid FileId) (FileHash, error)
	PersistData(file FileData) FileHash
	GetFileLet(file FileId) FileLet
	SetFileLet(file FileId, let FileLet)
	GetChangeHash(fid FileId) ChangeHash
	FileDeleted(fid FileId) bool
	FileExists(file FileData) bool
}


type Change interface {

}


func ChangeDataHash(data ChangeData) ChangeHash {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	enc.Encode(data)
	hsh := fnv.New128()
	sum := hsh.Sum(buf.Bytes());

	var ch ChangeHash
	for i:= 0; i < 16; i++ {
		ch.id[i] = sum[i]
	}
	return ch
}

type ChangeData struct {
	ID FileId
	Parent ChangeHash
	Data Change
}


type ItemCreated struct {
	Folder string
	Name string
	IsDirectory bool
}

type FileDataChanged struct {
	Size int64
	ModTime time.Time
}

type PersistedFileData struct {
	Hash FileHash
}

type FileDeleted struct {
	
}


func getFileUpdates(dc FilePersistence, file FileData) chan ChangeData{
	ch := make(chan ChangeData)
	go func(){
		id,err := dc.GetPersistId(file)
		if err != nil {
			id = dc.GenPersistId(file)
			var cd ChangeData
			cd.ID = id
			var ic ItemCreated
			cd.Data = ic
			ic.Name = file.Name
			ic.Folder = file.Folder
			ic.IsDirectory = file.IsDirectory
			ch <- cd
		}
		if dc.FileExists(file) == false && dc.FileDeleted(id) == false {
			var cd ChangeData
			cd.ID = id
			cd.Parent = dc.GetChangeHash(id)
			cd.Data = FileDeleted{}
			ch <- cd
			return;
		}

		
		flet := dc.GetFileLet(id);
		if flet.Size == file.Size && flet.ModTime == file.ModTime {
			return
		}

		nhsh := dc.PersistData(file)
		{
			var cd ChangeData
			cd.ID = id
			cd.Parent = dc.GetChangeHash(id)
			var fd PersistedFileData
			fd.Hash = nhsh
			ch <- cd
		}
		
		{
			flet := file.ToFileLet()
			var cd ChangeData
			cd.ID = id
			cd.Parent = dc.GetChangeHash(id)
			var fd FileDataChanged
			fd.Size = flet.Size
			fd.ModTime = flet.ModTime
			ch <- cd
		}
	}()
	return ch
}
