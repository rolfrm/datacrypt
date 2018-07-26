package main
// #include <errno.h>
// #include <poll.h>
// #include <stdio.h>
// #include <stdlib.h>
// #include <sys/inotify.h>
// #include <unistd.h>
//
//
// void fswatch_init(){
//   printf("Hello??\n");
// }
//
//
//
//
import "C"
import "unsafe"
import "fmt"
import "encoding/binary"

type FsConfig struct{
	fd _Ctype_int
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
	
	return fmt.Sprintf("%v %s %s: %s ", evt.wd, tp, action, evt.name) 

}

func FswatchInit(str string) FsConfig{
	C.fswatch_init();
	fd := C.inotify_init1(0);
	cstr:= C.CString(str)
	wd := C.inotify_add_watch(fd,cstr , C.IN_MODIFY | C.IN_CREATE | C.IN_DELETE | C.IN_DELETE_SELF | C.IN_MOVE_SELF | C.IN_MOVED_FROM | C.IN_MOVED_TO)
	C.free(unsafe.Pointer(cstr))
	fmt.Println("watch descriptor", wd)
	return FsConfig{fd: fd}
}


func FswatchPoll(fs FsConfig, stream chan INotifyEvent){
	var bytes [350]byte;
	read := C.read(fs.fd,unsafe.Pointer(&bytes[0]) , 350);
	fmt.Println(read);
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
		name := string(slice[:length])
		slice = slice[length:]
		thing := INotifyEvent{ wd: int(wd), mask: mask, cookie: cookie, name : name}
		stream <-  thing
		
	}
	
}
