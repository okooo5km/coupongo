# CouponGo Makefile

# 变量定义
BINARY_NAME=coupongo
CMD_PATH=./cmd/cli
BUILD_DIR=build
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-s -w -X main.version=${VERSION}"

# 默认目标
.PHONY: all
all: build

# 构建
.PHONY: build
build:
	@echo "Building ${BINARY_NAME}..."
	@mkdir -p ${BUILD_DIR}
	go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME} ${CMD_PATH}
	@echo "Built ${BUILD_DIR}/${BINARY_NAME}"

# 安装到系统
.PHONY: install
install: build
	@echo "Installing ${BINARY_NAME} to /usr/local/bin..."
	sudo cp ${BUILD_DIR}/${BINARY_NAME} /usr/local/bin/
	@echo "Installed successfully! Run '${BINARY_NAME} --help' to get started."

# 安装到用户目录
.PHONY: install-user
install-user: build
	@echo "Installing ${BINARY_NAME} to ~/.local/bin..."
	@mkdir -p ~/.local/bin
	cp ${BUILD_DIR}/${BINARY_NAME} ~/.local/bin/
	@chmod +x ~/.local/bin/${BINARY_NAME}
	@echo "Installed to ~/.local/bin/${BINARY_NAME}"
	@if ! echo "$$PATH" | grep -q "$$HOME/.local/bin"; then \
		echo ""; \
		echo "⚠️  Note: ~/.local/bin is not in your PATH."; \
		echo "Add it to your PATH by running:"; \
		echo "  echo 'export PATH=\"\$$HOME/.local/bin:\$$PATH\"' >> ~/.zshrc"; \
		echo "  source ~/.zshrc"; \
		echo ""; \
		echo "Or for bash:"; \
		echo "  echo 'export PATH=\"\$$HOME/.local/bin:\$$PATH\"' >> ~/.bashrc"; \
		echo "  source ~/.bashrc"; \
	else \
		echo "✅ ~/.local/bin is already in your PATH. You can run '${BINARY_NAME}' from anywhere."; \
	fi

# 交叉编译
.PHONY: build-all
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p ${BUILD_DIR}
	
	# Linux amd64
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-linux-amd64 ${CMD_PATH}
	
	# Linux arm64
	GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-linux-arm64 ${CMD_PATH}
	
	# macOS amd64
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-darwin-amd64 ${CMD_PATH}
	
	# macOS arm64 (Apple Silicon)
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-darwin-arm64 ${CMD_PATH}
	
	# Windows amd64
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-windows-amd64.exe ${CMD_PATH}
	
	@echo "Cross-compilation completed:"
	@ls -la ${BUILD_DIR}/

# 清理
.PHONY: clean
clean:
	@echo "Cleaning up..."
	rm -rf ${BUILD_DIR}
	go clean

# 测试
.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./...

# 代码格式化
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...

# 静态分析
.PHONY: vet
vet:
	@echo "Running go vet..."
	go vet ./...

# 依赖管理
.PHONY: tidy
tidy:
	@echo "Tidying dependencies..."
	go mod tidy

# 开发环境检查
.PHONY: check
check: fmt vet test
	@echo "All checks passed!"

# 发布准备
.PHONY: release
release: clean check build-all
	@echo "Release build completed!"

# 卸载
.PHONY: uninstall
uninstall:
	@echo "Uninstalling ${BINARY_NAME}..."
	@if [ -f /usr/local/bin/${BINARY_NAME} ]; then \
		sudo rm -f /usr/local/bin/${BINARY_NAME}; \
		echo "Removed from /usr/local/bin"; \
	fi
	@if [ -f ~/.local/bin/${BINARY_NAME} ]; then \
		rm -f ~/.local/bin/${BINARY_NAME}; \
		echo "Removed from ~/.local/bin"; \
	fi
	@echo "Uninstall completed."
	@echo "Note: Configuration file ~/.coupongo.json was not removed."
	@echo "Remove it manually if needed: rm ~/.coupongo.json"

# 显示帮助
.PHONY: help
help:
	@echo "CouponGo Build Commands:"
	@echo ""
	@echo "  make build        - Build the binary"
	@echo "  make install      - Install to /usr/local/bin (requires sudo)"
	@echo "  make install-user - Install to ~/.local/bin (user directory)"
	@echo "  make uninstall    - Remove installed binary"
	@echo "  make build-all    - Cross-compile for multiple platforms"
	@echo "  make test         - Run tests"
	@echo "  make fmt          - Format code"
	@echo "  make vet          - Run static analysis"
	@echo "  make tidy         - Clean up dependencies"
	@echo "  make check        - Run fmt, vet, and test"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make release      - Complete release build"
	@echo "  make help         - Show this help"