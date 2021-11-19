### Intro

This repository contains a simple reproducer of deadlocking a go process
using:

- cgo code which wraps a cpp library that throws exceptions under the hood
- [cgosymbolizer](https://github.com/ianlancetaylor/cgosymbolizer)
- continuous profiling using pprof, which uses SIGPROF under the hood

The culprit is a stack unwinding routine (`_Unwind_Backtrace`), which, at least in some observed implementations, invokes `dl_iterate_phdr` along the way.

Iterating the loaded shared objects involves holding a mutex, which in some implementations (e.g. `pthread_mutex_lock`) does so in a non reentrant manner, which could lead to deadlocks if a signal handler interrupts the locking routine and tries to grab the lock itself.

### Running

```shell
docker build -t cgosymbolizer-deadlock .
docker run -it --rm --cap-add=SYS_PTRACE cgosymbolizer-deadlock bash

# Inside container
$> ./cgosymbolizer-deadlock &

# Wait for the process to deadlock (it will stop printing at some point), then introspect it via gdb
$> gdb -p $!

# You should see something similar to:
(gdb) info threads
  Id   Target Id                                        Frame
* 1    Thread 0x7ff536e39740 (LWP 8) "cgosymbolizer-d"  runtime.futex () at /usr/local/go/src/runtime/sys_linux_amd64.s:520
  2    Thread 0x7ff5101a7700 (LWP 9) "cgosymbolizer-d"  runtime.futex () at /usr/local/go/src/runtime/sys_linux_amd64.s:520
  3    Thread 0x7ff50f806700 (LWP 10) "cgosymbolizer-d" runtime.futex () at /usr/local/go/src/runtime/sys_linux_amd64.s:520
  4    Thread 0x7ff50f005700 (LWP 11) "cgosymbolizer-d" __lll_lock_wait () at ../sysdeps/unix/sysv/linux/x86_64/lowlevellock.S:103
  5    Thread 0x7ff50e7c4700 (LWP 12) "cgosymbolizer-d" __lll_lock_wait () at ../sysdeps/unix/sysv/linux/x86_64/lowlevellock.S:103
  6    Thread 0x7ff50dfc3700 (LWP 13) "cgosymbolizer-d" runtime.futex () at /usr/local/go/src/runtime/sys_linux_amd64.s:520


(gdb) thread 4
[Switching to thread 4 (Thread 0x7ff50f005700 (LWP 11))]
#0  __lll_lock_wait () at ../sysdeps/unix/sysv/linux/x86_64/lowlevellock.S:103
103     ../sysdeps/unix/sysv/linux/x86_64/lowlevellock.S: No such file or directory.
(gdb) bt
#0  __lll_lock_wait () at ../sysdeps/unix/sysv/linux/x86_64/lowlevellock.S:103
#1  0x00007ff53732a7d1 in __GI___pthread_mutex_lock (mutex=0x7ff537370990 <_rtld_global+2352>) at ../nptl/pthread_mutex_lock.c:115
#2  0x00007ff536f71a7f in __GI___dl_iterate_phdr (callback=callback@entry=0x4bd620 <phdr_callback>, data=data@entry=0x7ff50f004880) at dl-iteratephdr.c:40
#3  0x00000000004bd8ac in backtrace_initialize (state=state@entry=0x7ff537347000, filename=filename@entry=0x1976430 "./cgosymbolizer-deadlock",
    descriptor=<optimized out>, error_callback=error_callback@entry=0x4be5d0 <errorCallback>, data=data@entry=0x7ff50f004b10,
    fileline_fn=fileline_fn@entry=0x7ff50f004918) at elf.c:4894
#4  0x00000000004bdaaa in fileline_initialize (state=state@entry=0x7ff537347000, error_callback=error_callback@entry=0x4be5d0 <errorCallback>,
    data=data@entry=0x7ff50f004b10) at fileline.c:261
#5  0x00000000004bdb92 in backtrace_pcinfo (state=0x7ff537347000, pc=140691166044593, callback=0x4be510 <callback>, error_callback=0x4be5d0 <errorCallback>,
    data=0x7ff50f004b10) at fileline.c:295
#6  0x00000000004be66d in cgoSymbolizer (parg=0x7ff50f004b10) at symbolizer.c:106
#7  0x000000000046200d in runtime.asmcgocall () at /usr/local/go/src/runtime/asm_amd64.s:795
#8  0x0000000000000000 in ?? ()
(gdb) frame 1
#1  0x00007ff53732a7d1 in __GI___pthread_mutex_lock (mutex=0x7ff537370990 <_rtld_global+2352>) at ../nptl/pthread_mutex_lock.c:115
115     ../nptl/pthread_mutex_lock.c: No such file or directory.
(gdb) p mutex.__data
$1 = {__lock = 2, __count = 0, __owner = 0, __nusers = 0, __kind = 1, __spins = 0, __elision = 0, __list = {__prev = 0x0, __next = 0x0}}

(gdb) thread 5
[Switching to thread 5 (Thread 0x7ff50e7c4700 (LWP 12))]
#0  __lll_lock_wait () at ../sysdeps/unix/sysv/linux/x86_64/lowlevellock.S:103
103     ../sysdeps/unix/sysv/linux/x86_64/lowlevellock.S: No such file or directory.
(gdb) bt
#0  __lll_lock_wait () at ../sysdeps/unix/sysv/linux/x86_64/lowlevellock.S:103
#1  0x00007ff53732a7d1 in __GI___pthread_mutex_lock (mutex=0x7ff537370990 <_rtld_global+2352>) at ../nptl/pthread_mutex_lock.c:115
#2  0x00007ff536f71a7f in __GI___dl_iterate_phdr (callback=0x7ff5370110b0, data=0xc00008b4f0) at dl-iteratephdr.c:40
#3  0x00007ff537012361 in _Unwind_Find_FDE () from /lib/x86_64-linux-gnu/libgcc_s.so.1
#4  0x00007ff53700ea43 in ?? () from /lib/x86_64-linux-gnu/libgcc_s.so.1
#5  0x00007ff53700fc20 in ?? () from /lib/x86_64-linux-gnu/libgcc_s.so.1
#6  0x00007ff537010928 in _Unwind_Backtrace () from /lib/x86_64-linux-gnu/libgcc_s.so.1
#7  0x00000000004be7ec in cgoTraceback (parg=0xc00008ba70, parg@entry=<error reading variable: value has been optimized out>) at traceback.c:82
#8  0x00000000004b2e06 in x_cgo_callers (sig=27, info=0xc00008bbf0, context=0xc00008bac0, cgoTraceback=<optimized out>, cgoCallers=<optimized out>,
    sigtramp=0x463de0 <runtime.sigtramp>) at gcc_traceback.c:42
#9  <signal handler called>
#10 0x00007ff53732a7c0 in __GI___pthread_mutex_lock (mutex=0x7ff537370990 <_rtld_global+2352>) at ../nptl/pthread_mutex_lock.c:115
#11 0x00007ff536f71a7f in __GI___dl_iterate_phdr (callback=0x7ff5370110b0, data=0x7ff50e7c3770) at dl-iteratephdr.c:40
#12 0x00007ff537012361 in _Unwind_Find_FDE () from /lib/x86_64-linux-gnu/libgcc_s.so.1
#13 0x00007ff53700ea43 in ?? () from /lib/x86_64-linux-gnu/libgcc_s.so.1
#14 0x00007ff53700fe5d in ?? () from /lib/x86_64-linux-gnu/libgcc_s.so.1
#15 0x00007ff537010391 in _Unwind_RaiseException () from /lib/x86_64-linux-gnu/libgcc_s.so.1
#16 0x00007ff53722eb27 in __cxa_throw () from /usr/lib/x86_64-linux-gnu/libstdc++.so.6
#17 0x0000000000403456 in throwAndCatch ()
#18 0x000000c00003a7d0 in ?? ()
#19 0x000000c00003a798 in ?? ()
#20 0x0000000000000000 in ?? ()
```