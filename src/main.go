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

type filelet struct{
	Size int64
	Modification_time time.Time
	IsDirectory bool 
}

func (fd * filedata) toFileLet() filelet{
	return filelet {Size: fd.size,
		Modification_time: fd.modification_time,
		IsDirectory: fd.isDirectory}
}

func cachedFileData(dc * datacrypt, filename string) filedata {
	uname := encodeAsBase64(filename);
	df := filepath.Join(dc.localFolder, uname);
	txt,_ := iou.ReadFile(df);
	if len(txt) == 0 {
		return filedata{};
	}
	
	ofile,_ := os.OpenFile(df, os.O_RDONLY, 0);
	defer ofile.Close()
	dec := gob.NewDecoder(ofile);

	var out filedata
	if nil == dec.Decode(&out) {
		return filedata{};
	}
	return out;
}

func getFileData(filename string) filedata {
	f,err := os.Stat(filename)
	if err != nil {
		return filedata{}
	}
	return filedata {
		modification_time: f.ModTime(),
		size: f.Size()}
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

				var out filedata
				dec := gob.NewDecoder(ofile);
				dec.Decode(&out);
				ofile.Close()

				if out.size != value.Size() || out.modification_time.Before(value.ModTime()) {
					dc.list.PushBack(fp);
				}
			}
		}
	}
}

type FileId struct {
	id [16]byte
}

func (thing *filedata) getFileId(dc * datacrypt) FileId{
	relFolder,_ := filepath.Rel(dc.dataFolder, thing.folder);
	hsh := fnv.New128()
	io.WriteString(hsh, relFolder);
	io.WriteString(hsh, thing.name);
	hshbytes := hsh.Sum(nil)
	var fileid FileId
	copy(hshbytes[:16], fileid.id[:16])
	return fileid
}

func (dc * datacrypt) dbPut(section []byte, name []byte , thing interface{}) error{
	db := dc.db;
	db.Update(func(tx * bolt.Tx) error{
		var buf bytes.Buffer
		enc := gob.NewEncoder(&buf)
		enc.Encode(thing)
		
		b := tx.Bucket(section)
		bytes := buf.Bytes()
		b.Put(name, bytes)
		return nil
	})
	return nil
}

func (dc * datacrypt) dbGet(section []byte, name []byte, thing interface{}) error{
	db := dc.db;
	ok := true
	err := db.View(func(tx * bolt.Tx) error{
		b := tx.Bucket(section)
		v := b.Get(name)
		if v != nil {
			buf := bytes.NewBuffer(v)
			dec := gob.NewDecoder(buf)
			err := dec.Decode(thing)
			if err != nil {
				ok = false
			}
		}else{
			ok = false
		}
		return nil
	})
	if err != nil {
		ok = false
	}
	if !ok {
		return fmt.Errorf("Unable to read item from db")
	}
	return nil
}

func (dc * datacrypt) dbEnsureBucket( name []byte){
	dc.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(name)
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})


}


func dbGetFileInfo(dc * datacrypt, thing FileId) (filelet, error){
	var result filelet
	err := dc.dbGet([]byte("files"), thing.id[:16], &result)
	return result, err
}

func dbSetFileInfo(dc * datacrypt, thing FileId, value filelet) error{
	err := dc.dbPut([]byte("files"), thing.id[:16], value)
	return err
}

func dataCryptRegister(dc *datacrypt, thing filedata){
	id := thing.getFileId(dc)
	fi, err := dbGetFileInfo(dc, id)
	
	if err == nil {
		if (fi.Size == thing.size) && (fi.Modification_time == thing.modification_time){
			return;
			
		}		
	}
	
		
}

func dataCryptInitialize(dc * datacrypt){
	db, err := bolt.Open(filepath.Join(dc.localFolder, "state.db"), 0600, nil)
	if err != nil {
		panic("Unable to open database")
	}
	dc.db = db

	dc.dbEnsureBucket([]byte("files"))
	
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
