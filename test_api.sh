#!/bin/bash

set -e

echo "=== 中央厨房追溯系统 API 测试脚本 ==="

BASE_URL="http://localhost:8080/api/v1"

echo ""
echo "1. 创建配方 (宫保鸡丁)"
curl -s -X POST "$BASE_URL/recipes" \
  -H "Content-Type: application/json" \
  -d '{
    "product_name": "宫保鸡丁",
    "product_code": "GBJD001",
    "items": [
      {"material_name": "鸡胸肉", "weight_per_unit": 0.2},
      {"material_name": "花生米", "weight_per_unit": 0.05},
      {"material_name": "干辣椒", "weight_per_unit": 0.02}
    ]
  }' | python3 -m json.tool

echo ""
echo "2. 原料入库 - 鸡胸肉 (供应商批次 CHK-20260601-001)"
curl -s -X POST "$BASE_URL/materials/inbound" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "鸡胸肉",
    "supplier": "正大食品",
    "supplier_batch_no": "CHK-20260601-001",
    "production_date": "2026-06-01",
    "shelf_life_days": 7,
    "weight": 100
  }' | python3 -m json.tool

echo ""
echo "3. 原料入库 - 鸡胸肉 (同一批次, 累加重量)"
curl -s -X POST "$BASE_URL/materials/inbound" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "鸡胸肉",
    "supplier": "正大食品",
    "supplier_batch_no": "CHK-20260601-001",
    "production_date": "2026-06-01",
    "shelf_life_days": 7,
    "weight": 50
  }' | python3 -m json.tool

echo ""
echo "4. 原料入库 - 花生米"
curl -s -X POST "$BASE_URL/materials/inbound" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "花生米",
    "supplier": "鲁花集团",
    "supplier_batch_no": "PEA-20260501-001",
    "production_date": "2026-05-01",
    "shelf_life_days": 180,
    "weight": 50
  }' | python3 -m json.tool

echo ""
echo "5. 原料入库 - 干辣椒"
curl -s -X POST "$BASE_URL/materials/inbound" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "干辣椒",
    "supplier": "贵州辣椒集团",
    "supplier_batch_no": "CHI-20260401-001",
    "production_date": "2026-04-01",
    "shelf_life_days": 365,
    "weight": 30
  }' | python3 -m json.tool

echo ""
echo "6. 创建生产工单 - 计划生产 100 份宫保鸡丁"
curl -s -X POST "$BASE_URL/workorders" \
  -H "Content-Type: application/json" \
  -d '{
    "product_name": "宫保鸡丁",
    "plan_quantity": 100
  }' | python3 -m json.tool

echo ""
echo "7. 完成工单 (请填入上方返回的 order_no)"
read -p "请输入工单号: " ORDER_NO
curl -s -X POST "$BASE_URL/workorders/$ORDER_NO/complete" \
  -H "Content-Type: application/json" \
  -d '{
    "actual_quantity": 100,
    "worker_ids": ["W001", "W002", "W003"],
    "inspection_result": "合格，口感、色泽均符合标准"
  }' | python3 -m json.tool

echo ""
echo "8. 门店签收 - 门店 S001"
read -p "请输入追溯码: " TRACE_CODE
curl -s -X POST "$BASE_URL/store/receipt" \
  -H "Content-Type: application/json" \
  -d "{
    \"trace_code\": \"$TRACE_CODE\",
    \"store_code\": \"S001\"
  }" | python3 -m json.tool

echo ""
echo "9. 正向追溯查询"
curl -s "$BASE_URL/trace/$TRACE_CODE" | python3 -m json.tool

echo ""
echo "10. 反向追溯查询 (鸡胸肉批次)"
curl -s "$BASE_URL/trace/reverse/material?supplier=正大食品&supplier_batch_no=CHK-20260601-001" | python3 -m json.tool

echo ""
echo "11. 库存看板"
curl -s "$BASE_URL/inventory/dashboard" | python3 -m json.tool

echo ""
echo "=== 测试完成 ==="
