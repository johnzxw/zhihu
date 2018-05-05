# zhihu

golang  echo 实现web版 知乎日报


安装： go get github.com/johnzxw/zhihu

编译命令： sh build.sh

启动命令：./zhihu (supervisor 启动)

数据库： sqlite3(zhihudb.db)


首页地址： ****:1101/

获取每天日报： ****:1101/getdata （请求:post，参数:date ,20180303）

