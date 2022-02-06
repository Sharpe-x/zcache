#!/bin/bash
trap "rm server;kill 0" EXIT


# command1 & command2 & command3 三个命令同时执行
# command1;command2;command3  不管前面命令执行成功没有，后面的命令继续执行
# command1 && command2 && command3  只有前面命令执行成功，后面命令才继续执行
go build -o server pickpeer.go
./server -port=8001 &
./server -port=8002 &
./server -port=8003 -api=1 &

sleep 3
echo ">>> start test"

#同时执行 
curl "http://127.0.0.1:9999/api?key=Tom" &
curl "http://127.0.0.1:9999/api?key=Tom" &
curl "http://127.0.0.1:9999/api?key=Tom" &


wait