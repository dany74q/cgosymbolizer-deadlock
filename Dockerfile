FROM golang:1.17-buster

RUN apt update && apt install gdb -y

WORKDIR /cgosymbolizer-deadlock
ADD . .

RUN gcc -c exceptions.cpp -o exceptions.o
RUN ar rcs exceptions.a exceptions.o

RUN go build .

CMD ["/cgosymbolizer-deadlock/cgosymbolizer-deadlock"]