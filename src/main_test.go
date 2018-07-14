package main
import "testing"
import "os"
import iou "io/ioutil"
import "strings"

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



func TestDataCrypt(t *testing.T){
	os.RemoveAll("data_test");
	os.RemoveAll("data_test_commits");
	
	os.Mkdir("data_test", 0777);
	os.Mkdir("data_test/test_dir", 0777);
	os.Mkdir("data_test_commits", 0777);
	iou.WriteFile("data_test/test1", make([]byte, 10), 0777)
	iou.WriteFile("data_test/test2", make([]byte, 20), 0777)
	iou.WriteFile("data_test/test_dir/test3", make([]byte, 30), 0777)

	fd := getFileData("data_test/test1");
	if fd.size != 10 {
		t.Errorf("unexpected size of data %d", fd.size);
	}
	
	dc :=  NewDataCrypt("data_test", "data_test_commits", "hello world");
	if dc == nil {
		t.Errorf("Unable to create dc");
	}
	dataCryptClose(dc);
	t.Log("Created dc");

	dc2 := NewDataCrypt("data_test", "data_test_commits", "hello world");
	if dc2 == nil {
		t.Errorf("Unable to create dc");
	}
	cachedFileData(dc2, "test1");

	{
		var f FileId
		f.id[1] = 1
		_,err := dbGetFileInfo(dc2, f)
		if err == nil {
			t.Errorf("Thing should not exist %v", err)
		}

		x := filelet {Size: 103}
		dbSetFileInfo(dc2, f, x)

		x3 := filelet {Size: 10}
		var f2 FileId
		f2.id[1] = 2
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
	val := 1234
	dc2.dbEnsureBucket([]byte("thing"))
	dc2.dbPut([]byte("thing"), []byte("asd"), val);
	val = 1
	dc2.dbGet([]byte("thing"), []byte("asd"), &val);
	if val != 1234 {
		//t.Errorf("Unable to deserialize correctly 1234 == %v", val)
	}
	dataCryptClose(dc2);
	os.RemoveAll(dc2.localFolder)
	//if test.initialized == false {
	//	t.Errorf("this should be initialized")
	//}
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
