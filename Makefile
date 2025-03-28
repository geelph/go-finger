.PHONY: build build-embed clean run test

# 默认目标
all: build

# 构建项目（不嵌入指纹库）
build:
	@echo "构建项目（不嵌入指纹库）..."
	@go build -o gxx ./cmd/main.go
	@echo "构建完成，请确保 'finger' 目录与二进制文件放在同一目录下"

# 构建项目（嵌入指纹库）
build-embed:
	@echo "构建项目（嵌入指纹库）..."
	@cp utils/finger/embed.go utils/finger/embed.go.bak
	@sed -i.bak 's|// //go:embed all:finger|//go:embed all:finger|g' utils/finger/embed.go
	@go build -o gxx ./cmd/main.go
	@mv utils/finger/embed.go.bak utils/finger/embed.go
	@echo "构建完成，指纹库已嵌入到二进制文件中"

# 使用build.sh脚本构建发布包
release:
	@echo "构建发布包（不嵌入指纹库）..."
	@chmod +x build.sh
	@./build.sh --all

# 使用build.sh脚本构建发布包（嵌入指纹库）
release-embed:
	@echo "构建发布包（嵌入指纹库）..."
	@chmod +x build.sh
	@./build.sh --embed --all

# 清理构建产物
clean:
	@echo "清理构建产物..."
	@rm -f gxx
	@rm -rf dist
	@echo "清理完成"

# 运行项目
run:
	@echo "运行项目..."
	@go run ./cmd/main.go

# 测试项目
test:
	@echo "运行测试..."
	@go test ./...

# 帮助信息
help:
	@echo "可用的命令:"
	@echo "  make build         - 构建项目（不嵌入指纹库）"
	@echo "  make build-embed   - 构建项目（嵌入指纹库）"
	@echo "  make release       - 构建发布包（不嵌入指纹库）"
	@echo "  make release-embed - 构建发布包（嵌入指纹库）"
	@echo "  make clean         - 清理构建产物"
	@echo "  make run           - 运行项目"
	@echo "  make test          - 运行测试"
	@echo "  make help          - 显示帮助信息" 