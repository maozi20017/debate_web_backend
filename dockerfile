# 使用官方 Go 鏡像作為基礎鏡像
FROM golang:1.22

# 設置工作目錄
WORKDIR /app

# 複製 go mod 和 sum 檔案
COPY go.mod go.sum ./

# 下載依賴
RUN go mod download

# 複製原始碼
COPY . .

# 編譯應用程式
RUN go build -o main

# 暴露埠口
EXPOSE 8080

# 執行應用程式
CMD ["./main"]