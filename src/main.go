package main

import "fmt"
import "os"
import "path/filepath"
import "hash/fnv"
import "io"
import "time"
import b64 "encoding/base32"
import str "strings"
import iou "io/ioutil"
import "crypto/rand"
import lst "container/list"
import "encoding/gob"
import "github.com/boltdb/bolt"
import "bytes"
import _ "compress/lzw"
import _ "crypto/aes"
import _ "crypto/cipher" // encrypt/decrypt

func printHelp(){
	fmt.Println("Usage: datacrypt [data folder] [commits folder] [encryption key]");
}

func encodeAsBase64(inputs ...string) string{
	hsh := fnv.New128()
	for _, str := range inputs {
		io.WriteString(hsh, str);
	}
	sum := hsh.Sum(nil);
	sEnc := b64.StdEncoding.EncodeToString(sum)
	return sEnc	
}

func encodeAsBase642(input []byte) string{
	hsh := fnv.New128()
	sum := hsh.Sum(input);
	sEnc := b64.StdEncoding.EncodeToString(sum)
	return sEnc	
}

func createUid() string{
	b := make([]byte, 32)
	rand.Read(b)
	return encodeAsBase642(b);
}

func fileExists(file string) bool{
	_, err := os.Stat(file)
	if err == nil {
		return true;
	}
	return false;
}

type datacrypt struct{
	dataFolder string
	localFolder string
	commitFolder string
	localName string
	list *lst.List
	key string
	db *bolt.DB
}

type FileLet struct{
	Size int64
	ModTime time.Time
	IsDirectory bool 
}

func (fd * FileData) ToFileLet() FileLet{
	return FileLet {Size: fd.Size,
		ModTime: fd.ModTime,
		IsDirectory: fd.IsDirectory}
}

func cachedFileData(dc * datacrypt, filename string) FileData {
	uname := encodeAsBase64(filename);
	df := filepath.Join(dc.localFolder, uname);
	txt,_ := iou.ReadFile(df);
	if len(txt) == 0 {
		return FileData{};
	}
	
	ofile,_ := os.OpenFile(df, os.O_RDONLY, 0);
	defer ofile.Close()
	dec := gob.NewDecoder(ofile);

	var out FileData
	if nil != dec.Decode(&out) {
		return FileData{};
	}
	return out;
}

func getFileData(filename string) FileData {
	f,err := os.Stat(filename)
	if err != nil {
		return FileData{}
	}
	return FileData {
		ModTime: f.ModTime(),
		Size: f.Size()}
}


// now iterate and compare timestamp and size.
// dont compare hash as it is too time consuming.
func initializeDirectory(dc * datacrypt, folder string){
	
	fmt.Println(folder);
	things,_:= iou.ReadDir(folder)
	for _,value := range things{
		fp := filepath.Join(folder, value.Name())
		if value.IsDir() {
			initializeDirectory(dc, fp)
		}

		uname := encodeAsBase64(fp);
		df := filepath.Join(dc.localFolder, uname);
		if !fileExists(df) {
			iou.WriteFile(df, make([]byte, 0) , 0777);
			dc.list.PushBack(fp);

		}else{
			txt,_ := iou.ReadFile(df);
			if len(txt) == 0 {
				
				dc.list.PushBack(fp);
			}else{
				
				fileid := string(txt)
				datafile := filepath.Join(dc.localFolder, fileid)
				ofile,_ := os.OpenFile(datafile, os.O_RDONLY, 0);

				var out FileData
				dec := gob.NewDecoder(ofile);
				dec.Decode(&out);
				ofile.Close()

				if out.Size != value.Size() || out.ModTime.Before(value.ModTime()) {
					dc.list.PushBack(fp);
				}
			}
		}
	}
}



func (thing *FileData) getFileId(dc * datacrypt) FileId{
	relFolder,_ := filepath.Rel(dc.dataFolder, thing.Folder);
	hsh := fnv.New128()
	io.WriteString(hsh, relFolder);
	io.WriteString(hsh, thing.Name);
	hshbytes := hsh.Sum(nil)
	var fileid FileId
	copy(fileid.ID[:16], hshbytes[:16])
	return fileid
}


func boltPut(db * bolt.DB,section []byte, name []byte , thing interface{}) error{
	return db.Update(func(tx * bolt.Tx) error{
		var buf bytes.Buffer
		if(thing != nil) {
			enc := gob.NewEncoder(&buf)
			err := enc.Encode(thing)
			if err != nil{
				return err;
			}
		}
		b := tx.Bucket(section)
		bytes := buf.Bytes()
		b.Put(name, bytes)
		return nil
	})
}

func boltGet(db * bolt.DB, section []byte, name []byte, thing interface{}) error{
	var innerErr error
	err := db.View(func(tx * bolt.Tx) error{
		b := tx.Bucket(section)
		v := b.Get(name)
		
		if v != nil && thing != nil {
			buf := bytes.NewBuffer(v)
			dec := gob.NewDecoder(buf)
			err := dec.Decode(thing)
			if err != nil {
				innerErr = err
			}
		}else{
			innerErr = fmt.Errorf("Unknown item %v", name)
		}
		return innerErr
	})
	
	return err
}


func (dc * datacrypt) dbPut(section []byte, name []byte , thing interface{}) error{
	return boltPut(dc.db, section, name, thing)
}



func (dc * datacrypt) dbGet(section []byte, name []byte, thing interface{}) error{
	return boltGet(dc.db, section, name, thing);
}

func boltEnsureBucket(db * bolt.DB, name []byte){
	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(name)
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
	
}

func (dc * datacrypt) dbEnsureBucket( name []byte){
	boltEnsureBucket(dc.db, name)

}


func dbGetFileInfo(dc * datacrypt, thing FileId) (FileLet, error){
	var result FileLet
	err := dc.dbGet([]byte("files"), thing.ID[:16], &result)
	return result, err
}

func dbSetFileInfo(dc * datacrypt, thing FileId, value FileLet) error{
	err := dc.dbPut([]byte("files"), thing.ID[:16], value)
	return err
}

func dataCryptRegister(dc *datacrypt, thing FileData){
	id := thing.getFileId(dc)
	fi, err := dbGetFileInfo(dc, id)
	
	if err == nil {
		if (fi.Size == thing.Size) && (fi.ModTime == thing.ModTime){
			return; // same as is already known
		}		
	}
	// thing does not exist in db, or only old version exists
	
	
	
		
}

func dataCryptInitialize(dc * datacrypt){
	db, err := bolt.Open(filepath.Join(dc.localFolder, "state.db"), 0600, nil)
	if err != nil {
		panic("Unable to open database")
	}
	dc.db = db

	dc.dbEnsureBucket([]byte("files"))
	dc.dbEnsureBucket([]byte("filehashes"))
	dc.dbEnsureBucket([]byte("lets"))
	dc.dbEnsureBucket([]byte("change"))
	
	localNameFile := filepath.Join(dc.localFolder, "machine");
	if !fileExists(localNameFile) {
		dc.localName = createUid()
		iou.WriteFile(localNameFile, []byte(dc.localName), 0777)
	}else{
		cname,_ := iou.ReadFile(localNameFile)
		dc.localName = string(cname)
	}

	for fd := range scanDirectory(dc.dataFolder) {
		dataCryptRegister(dc, fd)
	}
}

func dataCryptClose(dc * datacrypt){
	dc.db.Close()
}

func NewDataCrypt(dataFolder string, commitFolder string, key string) *datacrypt{
	dataFolderabs := dataFolder
	commitsFolderabs, _ := filepath.Abs(commitFolder)
	dataFolder = dataFolderabs;
	commitFolder = commitsFolderabs;
	if fileExists(dataFolder) == false {
		fmt.Println("Datafolder does not exist: ", dataFolder);
		return nil;
	}
	os.MkdirAll(commitFolder, 0777)
	var localFolder = str.Join([]string{encodeAsBase64(dataFolder), ".local"}, "")
	os.MkdirAll(localFolder, 0777);

	dc := new(datacrypt)
	dc.dataFolder = dataFolder;
	dc.localFolder = localFolder;
	dc.commitFolder = commitFolder;
	dc.list = lst.New();
	dc.key = key
	dataCryptInitialize(dc);
	return dc;
}

func DataCryptUpdate(dc *datacrypt){
	

}


func main() {
	args := os.Args[1:]
	if len(args) < 3 {
		printHelp()
		return
	}
	dc := NewDataCrypt(args[0], args[1], args[2]);
	fmt.Println(dc);
	
}

