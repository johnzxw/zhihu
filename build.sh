#!/usr/bin/env bash

#��ʼ����go���� -s ȥ�����ű� -w ȥ��DWARF������Ϣ
go build  -o zhihu -race -ldflags "-s -w"  main.go

#����upxѹ��
upx -9 zhihu