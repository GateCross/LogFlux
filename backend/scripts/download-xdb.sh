#!/bin/bash
# 下载 ip2region 最新 xdb 数据文件
# 用法: ./scripts/download-xdb.sh

set -e

DATA_DIR="$(cd "$(dirname "$0")/.." && pwd)/internal/middleware/data"
BASE_URL="https://raw.githubusercontent.com/lionsoul2014/ip2region/master/data"

mkdir -p "$DATA_DIR"

echo "下载 ip2region xdb 数据文件..."

for file in ip2region_v4.xdb ip2region_v6.xdb; do
    echo "  -> $file"
    curl -fsSL "$BASE_URL/$file" -o "$DATA_DIR/$file"
done

echo "下载完成: $DATA_DIR"
ls -lh "$DATA_DIR"/*.xdb
