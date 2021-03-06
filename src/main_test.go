package main
import "testing"
import "os"
import iou "io/ioutil"
import "strings"
import "crypto/aes"
import "crypto/sha256"
import "io"
import "crypto/cipher"
import compress "compress/zlib"
import "bytes"
import "github.com/boltdb/bolt"
import "encoding/json"
import "path/filepath"
import "time"
import "fmt"

func Test1(t *testing.T){
	
	file := "testfile"
	os.Remove(file)
	iou.WriteFile(file, make([]byte, 4) , 0666);
	if !fileExists(file){
		t.Errorf("File should exist '%s'", file)
	}

	if fileExists("testfile_not_exist"){
		t.Errorf("File should not exist..")
	}
	
	a,b := createUid(), createUid()
	if 0 == strings.Compare(a, b) {
		t.Errorf("Strings should not be equal %s == %s", a, b);
	}
}

func TestFsNotify(t * testing.T){
	os.RemoveAll("data_test2");
	os.RemoveAll("data_test2_commits");
	
	os.Mkdir("data_test2", 0777);
	os.Mkdir("data_test2/test_dir", 0777);
	os.Mkdir("data_test2/test_dir/test", 0777);
	os.Mkdir("data_test2_commits", 0777);
	iou.WriteFile("data_test2/test1", make([]byte, 10), 0777)
	iou.WriteFile("data_test2/test2", make([]byte, 20), 0777)
	iou.WriteFile("data_test2/test_dir/test3", make([]byte, 30), 0777)

	fs := FsWatchInit();
	fs.Add("data_test2");
	fs.Add("data_test2/test_dir");
	fs.Add("data_test2/test_dir/test");

	awaitDir := func(dir string){
		
		os.Mkdir(dir, 0777);
		for {
			select {
			case evt := <-fs.Events:
				if evt.IsDir() && strings.Compare(evt.Path() , dir) == 0 {
					fs.Add(evt.Path())
					return;
				}
			case <- time.After(1 * time.Second):
				panic("Timed out")
			}
		}
	}
	
	awaitRm := func(dir string){
		
		os.RemoveAll(dir);
		for {
			select {
			case evt := <-fs.Events:
				fmt.Println(evt, evt.IsDeleteSelf(), strings.Compare(evt.Path() , dir), evt.IsDir())
				if evt.IsDeleteSelf() {
				
					fs.Remove(evt.Path())
				}
				if evt.IsDeleteSelf() && strings.Compare(evt.Path() , dir) == 0 {
					return;
				}
			case <- time.After(1 * time.Second):
				panic("Timed out")
			}
		}

	}

	awaitWrite := func(file string, bytes int){
		fmt.Println("Written...")
		b := make([]byte, bytes)
		iou.WriteFile(file, b, 0777)
		created,modified := false,false
		for created == false || modified == false{
			select {
			case evt := <-fs.Events:
				fmt.Println(evt)
				if evt.IsCreate() {
					created = true
				}
				if evt.IsModify() {
					modified = true
				}
				
			case <- time.After(1 * time.Second):
				panic("Timed out")
			}
		}		
	}
	
	awaitDir("data_test2/test_dir/a")
	awaitDir("data_test2/test_dir/b")
	awaitDir("data_test2/test_dir/b/a")
	awaitDir("data_test2/test_dir/b/a/c")
	awaitDir("data_test2/test_dir/b/a/c/aaaaaaaaa")
	awaitDir("data_test2/test_dir/b/a/c/bbbbbbb")
	awaitWrite("data_test2/test_dir/b/a/asd", 10)
	awaitRm("data_test2/test_dir/b/a")
	fmt.Println("AUTOREMOVE")
	fs.AutoRemove = true
	os.RemoveAll("data_test2/test_dir")
	for fs.Count() > 1{
		select {
		case evt := <-fs.Events:
			fmt.Println(evt)
			
		case <- time.After(10 * time.Millisecond):
			continue;
		}
	}
			
	fmt.Println("DONE!!")
}

func TestDataCrypt(t *testing.T){

	
	fd := getFileData("data_test/test1");
	if fd.Size != 10 {
		t.Errorf("unexpected size of data %d", fd.Size);
	}
	
	dc :=  NewDataCrypt("data_test", "data_test_commits", "hello world");
	if dc == nil {
		t.Errorf("Unable to create dc");
	}
	
	iou.WriteFile("data_test/test_dir/test3", make([]byte, 35), 0777)
	
	dataCryptClose(dc);
	t.Log("Created dc");

	dc2 := NewDataCrypt("data_test", "data_test_commits", "hello world");
	if dc2 == nil {
		t.Errorf("Unable to create dc");
	}
	cachedFileData(dc2, "test1");

	{
		var f FileId
		f.ID[1] = 1
		_,err := dbGetFileInfo(dc2, f)
		if err == nil {
			t.Errorf("Thing should not exist %v", err)
		}

		x := FileLet {Size: 103}
		dbSetFileInfo(dc2, f, x)

		x3 := FileLet {Size: 10}
		var f2 FileId
		f2.ID[1] = 2
		dbSetFileInfo(dc2, f2, x3)
		dbGetFileInfo(dc2, f2)

		x2,err2 := dbGetFileInfo(dc2, f)
		if err2 != nil {
			t.Errorf("Thing should exist %v", err2)
		}
		if x != x2 {
			t.Errorf("What the hell?? %v %v", x, x2)
		}

		t.Log("Got:", x2)
	}
	scan := scanDirectory("data_test");
	for x := range(scan) {
		relFolder,_ := filepath.Rel(dc2.dataFolder, x.Folder);
		x.Folder = relFolder
		{
			fid, err := dc2.GetPersistId(x)
			if err != nil {
				fid = dc2.GenPersistId(x)
				t.Log("gen", fid, err)		
			}
		}
		{
			fid, err := dc2.GetPersistId(x)
			if err != nil {
				t.Errorf("Error happened %v id: %v", err, fid)
			}
			if x.IsDirectory {
				continue
			}
			
			_,err = dc2.FilePersisted(fid)
			if err != nil {
				t.Errorf("there should be an error")
			}
			hsh,err := dc2.PersistData(x)
			if err != nil {
				panic(err)
			}
			hsh2,err := dc2.PersistData(x)
			if err != nil{
				panic(err)
			}
			if false == bytes.Equal(hsh.Hash[:16], hsh2.Hash[:]) {
				panic("Hashes are not the same")
			}
			
				
			
		}

		
	}
	
	dataCryptClose(dc2);
	os.RemoveAll(dc2.localFolder)
	
}



func TestChannel(t * testing.T){
	messages := make(chan string)
	done := make(chan bool)
	
	go func()  {
		messages <- "hello!"
		messages <- "hello?"
		messages <- "hello!"
		messages <- "hello!"
		messages <- "end"
		done <- true
		close(messages);
	}()
	//for msg := range messages {
	//	t.Log("Messages", msg)
	//}

	gogogo := true;
	
	for gogogo {
		select {
		case msg := <-messages:
			t.Log("Message ", msg);
		case x := <-done:
			gogogo = false;
			t.Log("DONE ", x);
		}
	}
} 

func TestScan(t * testing.T){
	scan := scanDirectory(".");
	iterations := 0
	for fd := range scan {
		t.Log("File: ", fd);
		iterations += 1
		if iterations > 100 {
			break;
		}
	}
}

func TestDemoEncryption(t * testing.T){
	iname := "to_encrypt.bin"
	oname := "encrypted.bin"
	oname2 := "decrypted.bin"
	
	data,_ := iou.ReadFile("./main_test.go")
	
	iou.WriteFile(iname, data, 0777)
		
	hsh := sha256.New()
	io.WriteString(hsh, "hello")
	cryptkey := hsh.Sum(nil)

	block, err := aes.NewCipher(cryptkey)
	if err != nil {
		panic(err)
	}
	
	{ // write to file
		inFile, err := os.Open(iname)
		if err != nil {
			panic(err)
		}
		
		var iv [aes.BlockSize]byte
		stream := cipher.NewOFB(block, iv[:])
		outFile, err := os.OpenFile(oname, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		
		if err != nil {
			panic(err)
		}
		
		writer := &cipher.StreamWriter{S: stream, W: outFile}
		compressed,_ := compress.NewWriterLevel(writer, compress.BestSpeed);
		if _, err := io.Copy(compressed, inFile); err != nil {
			panic(err)
		}
		compressed.Close()
		outFile.Close()
		inFile.Close()
	}

	{ // read from file
		inFile, err := os.Open(oname)
		
		if err != nil {
			panic(err)
		}
	
		var iv [aes.BlockSize]byte
		stream := cipher.NewOFB(block, iv[:])
		outFile, err := os.OpenFile(oname2, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			panic(err)
		}

		reader := &cipher.StreamReader{S: stream, R: inFile}
		decompressed,_ := compress.NewReader(reader)
		if _, err := io.Copy(outFile, decompressed); err != nil {
			panic(err)
		}
		decompressed.Close()
		outFile.Close()
		inFile.Close()

	}
	decompressedData,_ := iou.ReadFile(oname2);
	if bytes.Equal(decompressedData, data) == false {
		t.Errorf("input and output not the same %v %v", decompressedData, data)
	}
	
	os.Remove(iname)
	os.Remove(oname)
	os.Remove(oname2)	
}



func TestDemoEncryption2(t * testing.T){
	iname := "to_encrypt.bin"
	oname := "encrypted.bin"
	oname2 := "decrypted.bin"
	
	data,_ := iou.ReadFile("./main_test.go")
	
	iou.WriteFile(iname, data, 0777)
	key := "Hello"
	{ // write to file
		inFile, err := os.Open(iname)

		if err != nil {
			panic(err)
		}
		outFile, err := os.OpenFile(oname, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			panic(err)
		}
		compressed := CompressionWriter(outFile, key);
		if _, err := io.Copy(compressed, inFile); err != nil {
			panic(err)
		}
		compressed.Close()
		outFile.Close()
		inFile.Close()
	}

	{ // read from file
		inFile, err := os.Open(oname)
		
		if err != nil {
			panic(err)
		}
		outFile, err := os.OpenFile(oname2, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			panic(err)
		}
		decompressed := CompressionReader(inFile, key)
		if _, err := io.Copy(outFile, decompressed); err != nil {
			panic(err)
		}
		decompressed.Close()
		outFile.Close()
		inFile.Close()

	}
	decompressedData,_ := iou.ReadFile(oname2);
	if bytes.Equal(decompressedData, data) == false {
		t.Errorf("input and output not the same %v %v", decompressedData, data)
	}
	
	os.Remove(iname)
	os.Remove(oname)
	os.Remove(oname2)	
}

func toJson(thing interface{}) string{
	bytes,_ := json.Marshal(thing)
	return string(bytes)

}

func _TestBoltBig(t * testing.T){
	var db *bolt.DB
	db, _ = bolt.Open("test_state.db", 0600, nil)
	defer db.Close()
	//defer os.Remove("test_state.db")

	n1 := []byte("test1")
	
	boltEnsureBucket(db, n1)
	{
		boltPut(db, n1, []byte("hej"), make([]byte, 10000000))
		var back []byte
		boltGet(db,n1, []byte("hej"), &back)
		if  len(back) != 10000000 {
			t.Errorf("Error does not match!")
		}
	}
	for i := 0; i < 0; i++ {
		boltPut(db, n1, []byte("hej"), make([]byte, 10000000 * i))
		var back []byte
		boltGet(db,n1, []byte("hej"), &back)
		if  len(back) != 10000000 * i {
			t.Errorf("Error does not match!")
		}
	}
	t.Log(toJson(db.Stats()))
}


func TestLink(t * testing.T){
	data := make([]byte, 100)
	data[10] = 5
	data[0] = 42
	
	iname := "hello_link"
	oname := "hello_link2"
	iou.WriteFile(iname, data, 0666)
	err:= os.Link(iname, oname)
	if err != nil {
		switch err.(type) {
		case *os.LinkError:
			t.Errorf("Link Error: %v", err.(*os.LinkError).Err)
		}
	}
	os.Remove(iname)
	data2,err := iou.ReadFile(oname)
	if err != nil{
		panic(err)
	}
	if bytes.Equal(data2, data) == false {
		t.Errorf("Does not work as expected!")
		return;
	}
	os.Remove(oname)

}


func TestIntConv(t * testing.T){
     var x uint64
     x = 100000000000
     var y int32
     y = int32(x)
     t.Log(y)
     // the important thing here is that there is no overflow.
     // the conversion should wrap around.

}

func TestFilePathData(t * testing.T){
	a := path2FileData("./main_test.go");
	abs,err := filepath.Abs("./main_test.go")
	if err != nil {
		panic(err)
	}
	b := path2FileData(abs)
	if strings.Compare(a.Name, b.Name) != 0 {
		t.Errorf("Names should be the same");		
	}
	
	
}
