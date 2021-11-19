package main

/*
#cgo LDFLAGS: exceptions.a

extern void throwAndCatch();

void pasten() {
	while (1) {
		throwAndCatch();
	}
}
*/
import "C"

import (
	"bytes"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"time"

	_ "github.com/ianlancetaylor/cgosymbolizer"
)

func main() {
	runtime.GOMAXPROCS(2)

	go C.pasten()

	for i := 0; i < 100_000; i++ {
		println("Loop #" + strconv.Itoa(i))

		_ = pprof.Lookup("goroutine").WriteTo(os.Stdout, 2)

		var buf bytes.Buffer
		_ = pprof.StartCPUProfile(&buf)
		time.Sleep(1 * time.Second)
		pprof.StopCPUProfile()
	}
}
