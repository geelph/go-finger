# HY 新一代基于yaml的指纹识别工具

## 编译

```shell
go mod tidy

CGO_ENABLED=0 GOARCH=arm64 GOOS=darwin go build -ldflags "-w -s" -o gxx main.go
chmod u+x ./StartsScan
./gxx -h
```

## 使用

```shell
./gxx -h
```