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


type FsConfig struct{
	fd _Ctype_int

}

func FswatchInit(str string) FsConfig{
	C.fswatch_init();
	fd := C.inotify_init1(C.IN_NONBLOCK);
	
	C.inotify_add_watch(fd,C.CString(str) , C.IN_OPEN | C.IN_CLOSE)
	return FsConfig{fd: fd}
}


func FswatchPoll(fs FsConfig){
	var bytes [128]byte;
	ok := C.read(fs.fd,unsafe.Pointer(&bytes[0]) , 128);
	fmt.Println(ok);
}
