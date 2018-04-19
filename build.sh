#!/usr/bin/env bash

#开始编译go程序 -s 去掉符号表 -w 去掉DWARF调试信息
go build  -o zhihu -race -ldflags "-s -w"  main.go

#开启upx压缩
upx -9 zhihu
