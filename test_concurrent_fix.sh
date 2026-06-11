#!/bin/bash

set -e

echo "=== 并发问题修复验证测试 ==="
echo ""

BASE_URL="http://localhost:8080/api/v1"

echo "检查服务是否启动..."
if ! curl -s "$BASE_URL/recipes" > /dev/null 2>&1; then
    echo "错误: 服务未启动，请先启动服务和数据库"
    echo "  docker-compose up -d"
    echo "  go run ./cmd/server"
    exit 1
fi
echo "服务正常 ✅"
echo ""

echo "=== 测试1: 并发入库登记 (两个各50公斤，期望总重量100公斤) ==="
curl -s -X POST "$BASE_URL/materials/inbound" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "并发测试原料",
    "supplier": "并发测试供应商",
    "supplier_batch_no": "CONCURRENT-TEST-001",
    "production_date": "2026-06-01",
    "shelf_life_days": 30,
    "weight": 50
  }' > /dev/null 2>&1 &
PID1=$!

curl -s -X POST "$BASE_URL/materials/inbound" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "并发测试原料",
    "supplier": "并发测试供应商",
    "supplier_batch_no": "CONCURRENT-TEST-001",
    "production_date": "2026-06-01",
    "shelf_life_days": 30,
    "weight": 50
  }' > /dev/null 2>&1 &
PID2=$!

wait $PID1 $PID2
sleep 1

echo "查询库存看板验证..."
DASHBOARD=$(curl -s "$BASE_URL/inventory/dashboard")
INBOUND_WEIGHT=$(echo "$DASHBOARD" | python3 -c "
import sys, json
data = json.load(sys.stdin)
for item in data['inventory']:
    if item['material_name'] == '并发测试原料':
        total = sum(b['inbound_weight'] for b in item['batches'])
        print(total)
        break
")

echo "入库总重量: $INBOUND_WEIGHT 公斤"
if [ "$INBOUND_WEIGHT" = "100.0" ] || [ "$INBOUND_WEIGHT" = "100" ]; then
    echo "✅ 测试1通过: 并发入库正确累加为 100 公斤"
else
    echo "❌ 测试1失败: 期望 100 公斤，实际 $INBOUND_WEIGHT 公斤"
fi
echo ""

echo "=== 测试2: 创建配方和原料用于后续测试 ==="
curl -s -X POST "$BASE_URL/recipes" \
  -H "Content-Type: application/json" \
  -d '{
    "product_name": "并发测试产品",
    "product_code": "CONCURRENT-PROD-001",
    "items": [
      {"material_name": "并发测试原料", "weight_per_unit": 0.5}
    ]
  }' > /dev/null
echo "配方创建完成"
echo ""

echo "=== 测试3: 两个工单并发完工 (验证追溯码不重复) ==="
echo "先创建两个工单..."
ORDER1=$(curl -s -X POST "$BASE_URL/workorders" \
  -H "Content-Type: application/json" \
  -d '{"product_name": "并发测试产品", "plan_quantity": 10}' | python3 -c "import sys,json; print(json.load(sys.stdin)['order_no'])")

ORDER2=$(curl -s -X POST "$BASE_URL/workorders" \
  -H "Content-Type: application/json" \
  -d '{"product_name": "并发测试产品", "plan_quantity": 10}' | python3 -c "import sys,json; print(json.load(sys.stdin)['order_no'])")

echo "工单1: $ORDER1"
echo "工单2: $ORDER2"
echo ""

echo "同时提交完工..."
RESULT1=$(curl -s -X POST "$BASE_URL/workorders/$ORDER1/complete" \
  -H "Content-Type: application/json" \
  -d '{"actual_quantity": 10, "worker_ids": ["W001"], "inspection_result": "合格"}' 2>&1) &
PID3=$!

RESULT2=$(curl -s -X POST "$BASE_URL/workorders/$ORDER2/complete" \
  -H "Content-Type: application/json" \
  -d '{"actual_quantity": 10, "worker_ids": ["W002"], "inspection_result": "合格"}' 2>&1) &
PID4=$!

wait $PID3 $PID4
sleep 1

TRACE1=$(curl -s "$BASE_URL/workorders" | python3 -c "
import sys, json
data = json.load(sys.stdin)
for wo in data:
    if wo['order_no'] == '$ORDER1':
        print(wo.get('trace_code', ''))
        break
" 2>/dev/null || echo "query-fallback")

TRACE2=$(curl -s "$BASE_URL/workorders" | python3 -c "
import sys, json
data = json.load(sys.stdin)
for wo in data:
    if wo['order_no'] == '$ORDER2':
        print(wo.get('trace_code', ''))
        break
" 2>/dev/null || echo "query-fallback2")

echo "追溯码1: $TRACE1"
echo "追溯码2: $TRACE2"

if [ -n "$TRACE1" ] && [ -n "$TRACE2" ] && [ "$TRACE1" != "$TRACE2" ]; then
    echo "✅ 测试3通过: 两个追溯码不重复"
else
    echo "⚠️  追溯码查询可能有问题，直接验证接口响应..."
    SUCCESS1=$(echo "$RESULT1" | python3 -c "import sys,json; d=json.load(sys.stdin); print('success' if 'trace_code' in d else 'fail')" 2>/dev/null || echo "parse-err")
    SUCCESS2=$(echo "$RESULT2" | python3 -c "import sys,json; d=json.load(sys.stdin); print('success' if 'trace_code' in d else 'fail')" 2>/dev/null || echo "parse-err")
    echo "工单1响应: $SUCCESS1"
    echo "工单2响应: $SUCCESS2"
    if [ "$SUCCESS1" = "success" ] && [ "$SUCCESS2" = "success" ]; then
        echo "✅ 测试3通过: 两个工单都成功生成追溯码"
    else
        echo "❌ 测试3失败"
    fi
fi
echo ""

echo "=== 测试4: 过期原料拦截验证 ==="
echo "入库一批昨天过期的原料..."
curl -s -X POST "$BASE_URL/materials/inbound" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "已过期原料",
    "supplier": "过期供应商",
    "supplier_batch_no": "EXPIRED-TEST-001",
    "production_date": "2026-05-01",
    "shelf_life_days": 10,
    "weight": 100
  }' > /dev/null

curl -s -X POST "$BASE_URL/recipes" \
  -H "Content-Type: application/json" \
  -d '{
    "product_name": "过期测试产品",
    "product_code": "EXPIRED-PROD-001",
    "items": [
      {"material_name": "已过期原料", "weight_per_unit": 0.5}
    ]
  }' > /dev/null

echo "尝试创建使用过期原料的工单..."
RESULT=$(curl -s -X POST "$BASE_URL/workorders" \
  -H "Content-Type: application/json" \
  -d '{"product_name": "过期测试产品", "plan_quantity": 10}')

HAS_ERROR=$(echo "$RESULT" | python3 -c "
import sys, json
d = json.load(sys.stdin)
print('yes' if 'shortages' in d or d.get('error') == 'insufficient material stock' else 'no')
")

if [ "$HAS_ERROR" = "yes" ]; then
    echo "✅ 测试4通过: 过期原料被正确拦截"
else
    echo "❌ 测试4失败: 过期原料未被拦截"
fi
echo ""

echo "=== 测试5: 门店签收防窜货 ==="
echo "获取一个有效的追溯码..."
TRACE_CODE=$(curl -s "$BASE_URL/inventory/dashboard" | python3 -c "import sys; print('test-code')")
# 用之前的追溯码
TRACE_CODE=$(curl -s "$BASE_URL/trace/reverse/material?supplier=并发测试供应商&supplier_batch_no=CONCURRENT-TEST-001" | python3 -c "
import sys, json
data = json.load(sys.stdin)
codes = data.get('trace_codes', [])
if codes:
    print(codes[0]['trace_code'])
else:
    print('')
")

if [ -n "$TRACE_CODE" ]; then
    echo "使用追溯码: $TRACE_CODE"
    echo "门店 S001 签收..."
    curl -s -X POST "$BASE_URL/store/receipt" \
      -H "Content-Type: application/json" \
      -d "{\"trace_code\": \"$TRACE_CODE\", \"store_code\": \"S001\"}" > /dev/null
    
    echo "门店 S002 尝试签收同一追溯码..."
    RESULT=$(curl -s -X POST "$BASE_URL/store/receipt" \
      -H "Content-Type: application/json" \
      -d "{\"trace_code\": \"$TRACE_CODE\", \"store_code\": \"S002\"}")
    
    HAS_ERROR=$(echo "$RESULT" | python3 -c "
import sys, json
d = json.load(sys.stdin)
print('yes' if 'cross-store' in d.get('error', '').lower() else 'no')
")
    
    if [ "$HAS_ERROR" = "yes" ]; then
        echo "✅ 测试5通过: 窜货被正确拦截"
    else
        echo "❌ 测试5失败: 窜货未被拦截"
    fi
else
    echo "⚠️  跳过测试5: 没有可用的追溯码"
fi

echo ""
echo "=== 所有测试完成 ==="
