#!/bin/bash

# 定義倉庫 (請修改為你的用戶名/倉庫名)
REPO="Yat-Muk/prism-v2"

# 1. 動態獲取最新版本號 (通過 GitHub API)
echo "正在檢查最新版本..."
LATEST_VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_VERSION" ]; then
    echo "錯誤: 無法獲取最新版本號，請檢查網絡或倉庫地址。"
    exit 1
fi

echo "發現最新版本: $LATEST_VERSION"

# 2. 檢測系統架構
ARCH=$(uname -m)
if [[ "$ARCH" == "x86_64" ]]; then
    FILE_ARCH="amd64"
elif [[ "$ARCH" == "aarch64" ]]; then
    FILE_ARCH="arm64"
else
    echo "錯誤: 不支持的架構 $ARCH"
    exit 1
fi

# 3. 構造下載鏈接
FILENAME="prism_${LATEST_VERSION}_linux_${FILE_ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/$LATEST_VERSION/$FILENAME"

# 4. 下載與安裝
echo "正在下載 Prism $LATEST_VERSION for $FILE_ARCH..."
wget -q --show-progress "$URL" -O prism.tar.gz

if [ $? -ne 0 ]; then
    echo "下載失敗！請檢查網絡連接或確認 Release 文件是否存在。"
    echo "嘗試訪問的 URL: $URL"
    exit 1
fi

echo "解壓安裝中..."
tar -xzf prism.tar.gz

# 賦予權限並移動
chmod +x prism
if mv prism /usr/local/bin/; then
    echo "安裝成功！"
else
    echo "移動文件失敗，嘗試使用 sudo..."
    sudo mv prism /usr/local/bin/
fi

# 清理臨時文件
rm prism.tar.gz checksums.txt README.md LICENSE 2>/dev/null

echo "==============================================="
echo " Prism $LATEST_VERSION 安裝完畢！"
echo " 請輸入 'prism' (或 'prism') 啟動程序"
echo "==============================================="
