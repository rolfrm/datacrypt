package main
// #include <errno.h>
// #include <poll.h>
// #include <stdio.h>
// #include <stdlib.h>
// #include <sys/inotify.h>
// #include <unistd.h>
// #include <sys/poll.h>
//
//
// int fspoll(int fd){
//    struct pollfd fds[1];
//    fds[0].fd = fd;
//    fds[0].events = POLLIN;
//    int n = poll(fds, 1, 100);
//    if(n == 0) return -1;
//    if((fds[0].events & POLLIN) > 0) return POLLIN;
//    return -1;
// }
//void print(const char * s){printf(">>>>  %s\n", s);}
//
//
//
import "C"
import "unsafe"
import "fmt"
import "encoding/binary"
import "path/filepath"
type FsConfig struct{
	fd _Ctype_int
	wd2dir map[int32]string
	dir2wd map[string]int32
	Events chan INotifyEvent
	AutoRemove bool
	running bool
}

//struct inotify_event {
//    int      wd;       /* Watch descriptor */
//    uint32_t mask;     /* Mask of events */
//    uint32_t cookie;   /* Unique cookie associating related
//                          events (for rename(2)) */
//    uint32_t len;      /* Size of name field */
//    char     name[];   /* Optional null-terminated name */
//};
// size: 4 + 4 + 4 + 4 + len= 16 + len

type INotifyEvent struct {
	wd int
	mask uint32
	cookie uint32
	name string
	folder string
}

func (evt INotifyEvent) IsDir() bool {
	return (evt.mask & C.IN_ISDIR) > 0
}

func (evt INotifyEvent) IsCreate() bool {
	return (evt.mask & C.IN_CREATE) > 0
}


func (evt INotifyEvent) IsModify() bool {
	return (evt.mask & C.IN_MODIFY) > 0
}

func (evt INotifyEvent) IsMove() bool {
	return (evt.mask & C.IN_MOVE) > 0
}

func (evt INotifyEvent) IsDelete() bool {
	return (evt.mask & C.IN_DELETE) > 0
}

func (evt INotifyEvent) IsDeleteSelf() bool {
	return (evt.mask & C.IN_DELETE_SELF) > 0
}

func (evt INotifyEvent) Path() string {
	return filepath.Join(evt.folder, evt.name)
}


func (evt INotifyEvent) String() string{
	tp := "File"
	if evt.IsDir() {
		tp = "Dir"
	}
	action := "??"
	if evt.IsCreate() {
		action = "create"
	}
	if evt.IsModify(){
		action = "modify"
	}
	if evt.IsMove() {
		action = "move"
	}
	if evt.IsDelete() {
		action = "delete"
	}
	if evt.IsDeleteSelf() {
		action = "delete self"
	}
	
	return fmt.Sprintf("%v %s %s: %s  ", evt.wd, tp, action, filepath.Join(evt.folder, evt.name)) 

}

func FsWatchInit() * FsConfig{
	
	fd := C.inotify_init1(0);
	cfg := FsConfig{fd: fd}
	cfg.wd2dir = make(map[int32]string)
	cfg.dir2wd = make(map[string]int32)
	
	c := &cfg;
	c.Events = make(chan INotifyEvent, 10)
	c.running = true
	go func(){
		for c.running {
			c.Poll()
		}
	}()
	return c
}

func (fs * FsConfig) Count() int{
	return len(fs.wd2dir)
}

func (fs * FsConfig) Add(str string) {
	
	_,ok := fs.dir2wd[str]
	if ok {
		return;
	}
	
	cstr:= C.CString(str)
	
	wd := C.inotify_add_watch(fs.fd,cstr , C.IN_MODIFY | C.IN_CREATE | C.IN_DELETE | C.IN_DELETE_SELF | C.IN_MOVE_SELF | C.IN_MOVED_FROM | C.IN_MOVED_TO | C.IN_MASK_ADD)

	_,ok = fs.wd2dir[int32(wd)]
	C.free(unsafe.Pointer(cstr))
	if wd == -1 {
		panic("!!")
	}
	fs.dir2wd[str] = int32(wd)
	fs.wd2dir[int32(wd)] = str
}

func (fs * FsConfig) Remove(path string){

	_,ok := fs.dir2wd[path]
	if !ok {
		return;
	}
	wd := fs.dir2wd[path]
	C.inotify_rm_watch(fs.fd, _Ctype_int(wd))
	delete(fs.dir2wd, path)
	delete(fs.wd2dir, wd)
}

func (fs *FsConfig) Poll(){
	bytes := make([]byte,400);
	read := C.read(fs.fd,unsafe.Pointer(&bytes[0]) , 350);
	if(read < 0){
		return
	}
	
	var slice = bytes[:read]
	for len(slice) > 0 {
		wd := binary.LittleEndian.Uint32(slice)
		slice = slice[4:]
		mask := binary.LittleEndian.Uint32(slice)
		slice = slice[4:]
		cookie := binary.LittleEndian.Uint32(slice)
		slice = slice[4:]
			length := binary.LittleEndian.Uint32(slice)
		slice = slice[4:]
		l := uint32(0)
		for l < length && slice[l] != 0 {
			l++
		}
			name := string(slice[:l])
		slice = slice[length:]
		thing := INotifyEvent{ wd: int(wd), mask: mask, cookie: cookie, name : name,
			folder: fs.wd2dir[int32(wd)]}
		fmt.Println(thing)
		if fs.AutoRemove && thing.IsDeleteSelf() {
			fs.Remove(thing.Path())
		}
			fs.Events <-  thing
		
	}
	
}
