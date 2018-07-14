package main
import "os"
import iou "io/ioutil"
import "path/filepath"
import "time"

type filedata struct {
	modification_time time.Time
	size int64
	folder string
	name string
	isDirectory bool
}


func stat2FileData(stat os.FileInfo, folder string) filedata{
	return filedata {
		modification_time: stat.ModTime(),
		size: stat.Size(),
		folder: folder,
		name: stat.Name(),
		isDirectory: stat.IsDir()}
}

func _scanDirectory(folder string, ch chan filedata){
	things,_:= iou.ReadDir(folder)
	for _,value := range things {
		ch <- stat2FileData(value, folder)
	}
	
	for _,value := range things {
		if value.IsDir() {
			fp := filepath.Join(folder, value.Name())
			_scanDirectory(fp, ch)
		}
	}
}

func scanDirectory(folder string) chan filedata{
	ch := make(chan filedata, 10)
	go func(){
		_scanDirectory(folder, ch)
		close(ch)
	}()
	return ch
}

