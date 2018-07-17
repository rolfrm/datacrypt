package main
import "os"
import iou "io/ioutil"
import "path/filepath"
import "time"

type FileData struct {
	ModTime time.Time
	Size int64
	Folder string
	Name string
	IsDirectory bool
}


func stat2FileData(stat os.FileInfo, folder string) FileData{
	return FileData {
		ModTime: stat.ModTime(),
		Size: stat.Size(),
		Folder: folder,
		Name: stat.Name(),
		IsDirectory: stat.IsDir()}
}

func _scanDirectory(folder string, ch chan FileData){
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

func scanDirectory(folder string) chan FileData{
	ch := make(chan FileData, 10)
	go func(){
		_scanDirectory(folder, ch)
		close(ch)
	}()
	return ch
}

