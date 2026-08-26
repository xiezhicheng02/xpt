#!/bin/bash
# 直接在 bencode 包目录下运行此脚本
PKG="github.com/xiezc/xpt/pkg/bencode"
REPORT_DIR="github.com/xiezc/reports/bencode/test_reports"
mkdir -p $REPORT_DIR

echo "===== 1. 运行单元测试 ====="
go test -v $PKG > $REPORT_DIR/unit_test.log
echo "单元测试完成，结果：$REPORT_DIR/unit_test.log"

echo "===== 2. 生成代码覆盖率报告 ====="
go test -coverprofile=$REPORT_DIR/cover.out $PKG
go tool cover -func=$REPORT_DIR/cover.out > $REPORT_DIR/coverage_summary.txt
go tool cover -html=$REPORT_DIR/cover.out -o $REPORT_DIR/coverage.html
echo "覆盖率报告完成：$REPORT_DIR/coverage_summary.txt / coverage.html"

echo "===== 3. 运行基准性能测试 ====="
go test -bench=. -benchmem -run=^$ $PKG > $REPORT_DIR/benchmark.txt
echo "基准测试完成：$REPORT_DIR/benchmark.txt"

echo "===== 4. 模糊测试（可选，默认关闭，需要可取消注释） ====="
# go test -fuzz=FuzzDecode -fuzztime=10s $PKG > $REPORT_DIR/fuzz_decode.log
# go test -fuzz=FuzzEncodeDecodeRoundTrip -fuzztime=10s $PKG > $REPORT_DIR/fuzz_roundtrip.log

echo ""
echo "✅ 所有报告已生成到 $REPORT_DIR 目录"
ls -lh $REPORT_DIR
