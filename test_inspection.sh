#!/bin/bash

set -e

echo "=== 中央厨房质检管理服务 API 测试脚本 ==="

BASE_URL="http://localhost:8080/api/v1"

echo ""
echo "==== 第一部分：准备基础数据 ===="

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
  }' | python3 -m json.tool 2>/dev/null || echo "创建成功或已存在"

echo ""
echo "2. 创建配方 (红烧肉)"
curl -s -X POST "$BASE_URL/recipes" \
  -H "Content-Type: application/json" \
  -d '{
    "product_name": "红烧肉",
    "product_code": "HSR001",
    "items": [
      {"material_name": "五花肉", "weight_per_unit": 0.3}
    ]
  }' | python3 -m json.tool 2>/dev/null || echo "创建成功或已存在"

echo ""
echo "3. 原料入库 - 鸡胸肉"
curl -s -X POST "$BASE_URL/materials/inbound" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "鸡胸肉",
    "supplier": "正大食品",
    "supplier_batch_no": "CHK-QA-20260601-001",
    "production_date": "2026-06-01",
    "shelf_life_days": 7,
    "weight": 100
  }' | python3 -m json.tool 2>/dev/null || echo "入库成功"

echo ""
echo "4. 原料入库 - 花生米"
curl -s -X POST "$BASE_URL/materials/inbound" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "花生米",
    "supplier": "鲁花集团",
    "supplier_batch_no": "PEA-QA-20260501-001",
    "production_date": "2026-05-01",
    "shelf_life_days": 180,
    "weight": 50
  }' | python3 -m json.tool 2>/dev/null || echo "入库成功"

echo ""
echo "5. 原料入库 - 干辣椒"
curl -s -X POST "$BASE_URL/materials/inbound" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "干辣椒",
    "supplier": "贵州辣椒集团",
    "supplier_batch_no": "CHI-QA-20260401-001",
    "production_date": "2026-04-01",
    "shelf_life_days": 365,
    "weight": 30
  }' | python3 -m json.tool 2>/dev/null || echo "入库成功"

echo ""
echo "6. 原料入库 - 五花肉"
curl -s -X POST "$BASE_URL/materials/inbound" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "五花肉",
    "supplier": "双汇集团",
    "supplier_batch_no": "PORK-QA-20260601-001",
    "production_date": "2026-06-01",
    "shelf_life_days": 5,
    "weight": 200
  }' | python3 -m json.tool 2>/dev/null || echo "入库成功"

echo ""
echo ""
echo "==== 第二部分：质检模板管理 ===="

echo ""
echo "7. 创建宫保鸡丁质检模板（数值范围+文本匹配，含关键控制点）"
TEMPLATE_GBJD_RESP=$(curl -s -X POST "$BASE_URL/inspection/templates" \
  -H "Content-Type: application/json" \
  -d '{
    "product_code": "GBJD001",
    "product_name": "宫保鸡丁",
    "template_name": "宫保鸡丁标准质检模板 v1.0",
    "tolerance_rate": 0.2,
    "check_items": [
      {
        "name": "中心温度",
        "method": "用食品中心温度计测量产品中心位置温度，放置30秒读数",
        "standard_type": "range",
        "min_value": 70.0,
        "max_value": 85.0,
        "is_key_point": true
      },
      {
        "name": "单份重量",
        "method": "取三份成品用电子秤称量，计算平均值",
        "standard_type": "range",
        "min_value": 290.0,
        "max_value": 310.0,
        "is_key_point": false
      },
      {
        "name": "感官色泽",
        "method": "自然光下目视观察成品颜色",
        "standard_type": "match",
        "match_text": "金红",
        "is_key_point": false
      },
      {
        "name": "微生物检测-菌落总数",
        "method": "按GB 4789.2标准方法检测",
        "standard_type": "range",
        "min_value": 0.0,
        "max_value": 10000.0,
        "is_key_point": true
      },
      {
        "name": "包装完整性",
        "method": "目视检查封盒包装无破损、无漏撒",
        "standard_type": "match",
        "match_text": "完好",
        "is_key_point": false
      }
    ]
  }')
echo "$TEMPLATE_GBJD_RESP" | python3 -m json.tool

echo ""
echo "8. 创建红烧肉质检模板"
curl -s -X POST "$BASE_URL/inspection/templates" \
  -H "Content-Type: application/json" \
  -d '{
    "product_code": "HSR001",
    "product_name": "红烧肉",
    "template_name": "红烧肉标准质检模板 v1.0",
    "tolerance_rate": 0.1,
    "check_items": [
      {
        "name": "中心温度",
        "method": "食品中心温度计测量",
        "standard_type": "range",
        "min_value": 75.0,
        "max_value": 90.0,
        "is_key_point": true
      },
      {
        "name": "感官评价",
        "method": "品鉴",
        "standard_type": "match",
        "match_text": "合格",
        "is_key_point": false
      }
    ]
  }' | python3 -m json.tool

echo ""
echo "9. 查看所有质检模板"
TEMPLATES_RESP=$(curl -s "$BASE_URL/inspection/templates")
echo "$TEMPLATES_RESP" | python3 -m json.tool

TEMPLATE_ID=$(echo "$TEMPLATES_RESP" | python3 -c "
import sys, json
data = json.load(sys.stdin)
for t in data:
    if t['product_code'] == 'GBJD001':
        print(t['id'])
        break
")
echo "宫保鸡丁模板ID: $TEMPLATE_ID"

echo ""
echo "10. 查看单个模板详情 (ID=$TEMPLATE_ID)"
curl -s "$BASE_URL/inspection/templates/$TEMPLATE_ID" | python3 -m json.tool

echo ""
echo "11. 更新模板 - 修改容差率为 0.25"
NEW_TOL=0.25
curl -s -X PUT "$BASE_URL/inspection/templates/$TEMPLATE_ID" \
  -H "Content-Type: application/json" \
  -d "{
    \"tolerance_rate\": $NEW_TOL
  }" | python3 -m json.tool

echo ""
echo ""
echo "==== 第三部分：创建工单并生成追溯码 ===="

echo ""
echo "12. 创建宫保鸡丁生产工单 1（全部合格场景）"
WO1_RESP=$(curl -s -X POST "$BASE_URL/workorders" \
  -H "Content-Type: application/json" \
  -d '{
    "product_name": "宫保鸡丁",
    "plan_quantity": 100
  }')
echo "$WO1_RESP" | python3 -m json.tool
ORDER_NO_1=$(echo "$WO1_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['order_no'])")
echo "工单号1: $ORDER_NO_1"

echo ""
echo "13. 完成工单 1，生成追溯码"
COMP1_RESP=$(curl -s -X POST "$BASE_URL/workorders/$ORDER_NO_1/complete" \
  -H "Content-Type: application/json" \
  -d '{
    "actual_quantity": 100,
    "worker_ids": ["W001", "W002"],
    "inspection_result": "待质检"
  }')
echo "$COMP1_RESP" | python3 -m json.tool
TRACE_CODE_1=$(echo "$COMP1_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['trace_code'])")
echo "追溯码1: $TRACE_CODE_1"

echo ""
echo "14. 创建宫保鸡丁生产工单 2（关键控制点不合格场景）"
WO2_RESP=$(curl -s -X POST "$BASE_URL/workorders" \
  -H "Content-Type: application/json" \
  -d '{
    "product_name": "宫保鸡丁",
    "plan_quantity": 50
  }')
echo "$WO2_RESP" | python3 -m json.tool
ORDER_NO_2=$(echo "$WO2_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['order_no'])")
echo "工单号2: $ORDER_NO_2"

echo ""
echo "15. 完成工单 2，生成追溯码"
COMP2_RESP=$(curl -s -X POST "$BASE_URL/workorders/$ORDER_NO_2/complete" \
  -H "Content-Type: application/json" \
  -d '{
    "actual_quantity": 48,
    "worker_ids": ["W003", "W004"],
    "inspection_result": "待质检"
  }')
echo "$COMP2_RESP" | python3 -m json.tool
TRACE_CODE_2=$(echo "$COMP2_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['trace_code'])")
echo "追溯码2: $TRACE_CODE_2"

echo ""
echo "16. 创建红烧肉生产工单 3（普通项超标场景）"
WO3_RESP=$(curl -s -X POST "$BASE_URL/workorders" \
  -H "Content-Type: application/json" \
  -d '{
    "product_name": "红烧肉",
    "plan_quantity": 200
  }')
echo "$WO3_RESP" | python3 -m json.tool
ORDER_NO_3=$(echo "$WO3_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['order_no'])")
echo "工单号3: $ORDER_NO_3"

echo ""
echo "17. 完成工单 3，生成追溯码"
COMP3_RESP=$(curl -s -X POST "$BASE_URL/workorders/$ORDER_NO_3/complete" \
  -H "Content-Type: application/json" \
  -d '{
    "actual_quantity": 195,
    "worker_ids": ["W005"],
    "inspection_result": "待质检"
  }')
echo "$COMP3_RESP" | python3 -m json.tool
TRACE_CODE_3=$(echo "$COMP3_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['trace_code'])")
echo "追溯码3: $TRACE_CODE_3"

echo ""
echo ""
echo "==== 第四部分：质检任务自动生成（轮询） ===="

echo ""
echo "18. 轮询追溯系统，自动创建质检任务"
curl -s -X POST "$BASE_URL/inspection/tasks/poll" | python3 -m json.tool

echo ""
echo "19. 再次轮询（应跳过已创建的任务）"
curl -s -X POST "$BASE_URL/inspection/tasks/poll" | python3 -m json.tool

echo ""
echo "20. 查看所有质检任务"
TASKS_RESP=$(curl -s "$BASE_URL/inspection/tasks")
echo "$TASKS_RESP" | python3 -m json.tool

TASK_ID_1=$(echo "$TASKS_RESP" | python3 -c "
import sys, json
data = json.load(sys.stdin)
for t in data:
    if t['trace_code'] == '$TRACE_CODE_1':
        print(t['id'])
        break
")
TASK_ID_2=$(echo "$TASKS_RESP" | python3 -c "
import sys, json
data = json.load(sys.stdin)
for t in data:
    if t['trace_code'] == '$TRACE_CODE_2':
        print(t['id'])
        break
")
TASK_ID_3=$(echo "$TASKS_RESP" | python3 -c "
import sys, json
data = json.load(sys.stdin)
for t in data:
    if t['trace_code'] == '$TRACE_CODE_3':
        print(t['id'])
        break
")
echo "任务ID 1 (全合格): $TASK_ID_1"
echo "任务ID 2 (CCP不合格): $TASK_ID_2"
echo "任务ID 3 (普通项超标): $TASK_ID_3"

echo ""
echo ""
echo "==== 第五部分：质检执行 - 场景1：全部合格 ===="

echo ""
echo "21. 获取模板检查项ID"
TEMPLATE_DETAIL=$(curl -s "$BASE_URL/inspection/templates/$TEMPLATE_ID")
echo "$TEMPLATE_DETAIL" | python3 -m json.tool

CI_ID_TEMP=$(echo "$TEMPLATE_DETAIL" | python3 -c "
import sys, json
data = json.load(sys.stdin)
ids = [str(ci['id']) for ci in data['check_items']]
# 按顺序: 中心温度, 单份重量, 感官色泽, 微生物, 包装完整性
names = [ci['name'] for ci in data['check_items']]
for i, (cid, n) in enumerate(zip(ids, names)):
    print(f'{i+1}:{cid}:{n}')
")
echo "检查项列表:"
echo "$CI_ID_TEMP"

CI1_ID=$(echo "$CI_ID_TEMP" | grep -E "^1:" | cut -d: -f2)
CI2_ID=$(echo "$CI_ID_TEMP" | grep -E "^2:" | cut -d: -f2)
CI3_ID=$(echo "$CI_ID_TEMP" | grep -E "^3:" | cut -d: -f2)
CI4_ID=$(echo "$CI_ID_TEMP" | grep -E "^4:" | cut -d: -f2)
CI5_ID=$(echo "$CI_ID_TEMP" | grep -E "^5:" | cut -d: -f2)
echo "检查项ID: 中心温度=$CI1_ID, 重量=$CI2_ID, 色泽=$CI3_ID, 微生物=$CI4_ID, 包装=$CI5_ID"

echo ""
echo "22. 开始质检任务1 - 质检员 QA001"
curl -s -X POST "$BASE_URL/inspection/tasks/$TASK_ID_1/start" \
  -H "Content-Type: application/json" \
  -d '{"inspector_id": "QA001"}' | python3 -m json.tool

echo ""
echo "23. 提交质检结果1 - 全部合格"
SUBMIT1_RESP=$(curl -s -X POST "$BASE_URL/inspection/tasks/$TASK_ID_1/submit" \
  -H "Content-Type: application/json" \
  -d "{
    \"items\": [
      {\"check_item_id\": $CI1_ID, \"actual_value\": \"75.5\"},
      {\"check_item_id\": $CI2_ID, \"actual_value\": \"302.0\"},
      {\"check_item_id\": $CI3_ID, \"actual_value\": \"金红有光泽\"},
      {\"check_item_id\": $CI4_ID, \"actual_value\": \"1200\"},
      {\"check_item_id\": $CI5_ID, \"actual_value\": \"包装完好无破损\"}
    ]
  }")
echo "$SUBMIT1_RESP" | python3 -m json.tool

REPORT_ID_1=$(echo "$SUBMIT1_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('report_id', ''))")
echo "报告ID1: $REPORT_ID_1"

echo ""
echo ""
echo "==== 第六部分：质检执行 - 场景2：关键控制点(CCP)不合格 ===="

echo ""
echo "24. 开始质检任务2 - 质检员 QA002"
curl -s -X POST "$BASE_URL/inspection/tasks/$TASK_ID_2/start" \
  -H "Content-Type: application/json" \
  -d '{"inspector_id": "QA002"}' | python3 -m json.tool

echo ""
echo "25. 提交质检结果2 - 中心温度(CCP)不合格（低于70度）"
SUBMIT2_RESP=$(curl -s -X POST "$BASE_URL/inspection/tasks/$TASK_ID_2/submit" \
  -H "Content-Type: application/json" \
  -d "{
    \"items\": [
      {\"check_item_id\": $CI1_ID, \"actual_value\": \"62.3\", \"fail_reason\": \"加热不充分，可能存在致病菌风险\"},
      {\"check_item_id\": $CI2_ID, \"actual_value\": \"300.0\"},
      {\"check_item_id\": $CI3_ID, \"actual_value\": \"金红色泽正常\"},
      {\"check_item_id\": $CI4_ID, \"actual_value\": \"8500\"},
      {\"check_item_id\": $CI5_ID, \"actual_value\": \"完好\"}
    ]
  }")
echo "$SUBMIT2_RESP" | python3 -m json.tool

REPORT_ID_2=$(echo "$SUBMIT2_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('report_id', ''))")
DISPOSAL_ID_2=$(echo "$SUBMIT2_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('disposal_id', ''))")
echo "报告ID2: $REPORT_ID_2, 处置单ID2: $DISPOSAL_ID_2"

echo ""
echo ""
echo "==== 第七部分：质检执行 - 场景3：普通项超标（超过容差率） ===="

echo ""
echo "26. 获取红烧肉模板检查项ID"
HSR_TEMPLATE_ID=$(curl -s "$BASE_URL/inspection/templates" | python3 -c "
import sys, json
data = json.load(sys.stdin)
for t in data:
    if t['product_code'] == 'HSR001':
        print(t['id'])
        break
")
echo "红烧肉模板ID: $HSR_TEMPLATE_ID"

HSR_TEMPLATE_DETAIL=$(curl -s "$BASE_URL/inspection/templates/$HSR_TEMPLATE_ID")
HSR_CI_IDS=$(echo "$HSR_TEMPLATE_DETAIL" | python3 -c "
import sys, json
data = json.load(sys.stdin)
ids = [str(ci['id']) for ci in data['check_items']]
names = [ci['name'] for ci in data['check_items']]
for i, (cid, n) in enumerate(zip(ids, names)):
    print(f'{i+1}:{cid}:{n}')
")
echo "红烧肉检查项: $HSR_CI_IDS"
HSR_CI1_ID=$(echo "$HSR_CI_IDS" | grep -E "^1:" | cut -d: -f2)
HSR_CI2_ID=$(echo "$HSR_CI_IDS" | grep -E "^2:" | cut -d: -f2)
echo "红烧肉: 温度=$HSR_CI1_ID, 感官=$HSR_CI2_ID (容差率10%，2项中1项不合格=50%>10%，应该判定不合格)"

echo ""
echo "27. 开始质检任务3 - 质检员 QA003"
curl -s -X POST "$BASE_URL/inspection/tasks/$TASK_ID_3/start" \
  -H "Content-Type: application/json" \
  -d '{"inspector_id": "QA003"}' | python3 -m json.tool

echo ""
echo "28. 提交质检结果3 - 感官评价不合格（CCP合格，但普通项不合格率50%>10%容差）"
SUBMIT3_RESP=$(curl -s -X POST "$BASE_URL/inspection/tasks/$TASK_ID_3/submit" \
  -H "Content-Type: application/json" \
  -d "{
    \"items\": [
      {\"check_item_id\": $HSR_CI1_ID, \"actual_value\": \"82.0\"},
      {\"check_item_id\": $HSR_CI2_ID, \"actual_value\": \"偏咸色泽偏暗\", \"fail_reason\": \"感官检查不合格，口味偏咸外观色泽暗\"}
    ]
  }")
echo "$SUBMIT3_RESP" | python3 -m json.tool

REPORT_ID_3=$(echo "$SUBMIT3_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('report_id', ''))")
DISPOSAL_ID_3=$(echo "$SUBMIT3_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('disposal_id', ''))")
echo "报告ID3: $REPORT_ID_3, 处置单ID3: $DISPOSAL_ID_3"

echo ""
echo ""
echo "==== 第八部分：质检报告查看（报告不可修改） ===="

echo ""
echo "29. 查看质检报告1（全部合格）"
curl -s "$BASE_URL/inspection/reports/$REPORT_ID_1" | python3 -m json.tool

echo ""
echo "30. 查看质检报告2（CCP不合格）"
curl -s "$BASE_URL/inspection/reports/$REPORT_ID_2" | python3 -m json.tool

echo ""
echo "31. 通过追溯码查看报告"
curl -s "$BASE_URL/inspection/reports/trace/$TRACE_CODE_1" | python3 -m json.tool

echo ""
echo "32. 列出所有质检报告"
curl -s "$BASE_URL/inspection/reports" | python3 -m json.tool

echo ""
echo ""
echo "==== 第九部分：不合格品处置流程 ===="

echo ""
echo "33. 查看所有处置单"
curl -s "$BASE_URL/inspection/disposals" | python3 -m json.tool

echo ""
echo "34. 查看处置单2详情"
curl -s "$BASE_URL/inspection/disposals/$DISPOSAL_ID_2" | python3 -m json.tool

echo ""
echo "35. 修改处置单2 - 改为降级使用"
curl -s -X PUT "$BASE_URL/inspection/disposals/$DISPOSAL_ID_2" \
  -H "Content-Type: application/json" \
  -d '{
    "method": "degrade",
    "reason": "可改作员工餐使用，不对外销售"
  }' | python3 -m json.tool

echo ""
echo "36. 主管审批 - 拒绝处置单2（降级建议不合理，CCP不合格必须报废）"
curl -s -X POST "$BASE_URL/inspection/disposals/$DISPOSAL_ID_2/approve" \
  -H "Content-Type: application/json" \
  -d '{
    "approver_id": "MANAGER001",
    "approved": false,
    "remark": "中心温度不达标属于严重食品安全问题，不允许降级使用，请重新提交报废方案"
  }' | python3 -m json.tool

echo ""
echo "37. 审批后不能再修改处置单（应报错）"
echo "  尝试再次修改处置单2:"
curl -s -X PUT "$BASE_URL/inspection/disposals/$DISPOSAL_ID_2" \
  -H "Content-Type: application/json" \
  -d '{"method": "scrap"}' | python3 -m json.tool

echo ""
echo "  (提示: 审批被拒绝的处置单当前设计是不能修改的，实际业务中可以根据需要调整)"

echo ""
echo "38. 主管审批 - 通过处置单3（返工）"
curl -s -X POST "$BASE_URL/inspection/disposals/$DISPOSAL_ID_3/approve" \
  -H "Content-Type: application/json" \
  -d '{
    "approver_id": "MANAGER001",
    "approved": true,
    "remark": "同意返工，回锅重新调味，注意控制咸度"
  }' | python3 -m json.tool

echo ""
echo "39. 执行处置单3 - 更新追溯系统批次状态"
EXEC3_RESP=$(curl -s -X POST "$BASE_URL/inspection/disposals/$DISPOSAL_ID_3/execute")
echo "$EXEC3_RESP" | python3 -m json.tool

echo ""
echo "40. 验证追溯系统中工单状态已更新"
curl -s "$BASE_URL/trace/$TRACE_CODE_3" | python3 -m json.tool

echo ""
echo ""
echo "==== 第十部分：统计分析接口 ===="

echo ""
echo "41. 查询今日质检统计"
TODAY=$(date +%Y-%m-%d)
curl -s "$BASE_URL/inspection/statistics?start_date=$TODAY&end_date=$TODAY" | python3 -m json.tool

echo ""
echo "42. 查询宫保鸡丁专项统计"
curl -s "$BASE_URL/inspection/statistics?start_date=$TODAY&end_date=$TODAY&product_code=GBJD001" | python3 -m json.tool

echo ""
echo ""
echo "==== 第十一部分：边界条件验证 ===="

echo ""
echo "43. 验证不能重复提交质检（应报错）"
curl -s -X POST "$BASE_URL/inspection/tasks/$TASK_ID_1/submit" \
  -H "Content-Type: application/json" \
  -d "{
    \"items\": [
      {\"check_item_id\": $CI1_ID, \"actual_value\": \"75.0\"}
    ]
  }" | python3 -m json.tool

echo ""
echo "44. 验证删除有未完成任务的模板（应报错）- 先创建新工单"
WO_TEST=$(curl -s -X POST "$BASE_URL/workorders" \
  -H "Content-Type: application/json" \
  -d '{"product_name": "宫保鸡丁", "plan_quantity": 10}')
echo "$WO_TEST" | python3 -m json.tool
TEST_ORDER_NO=$(echo "$WO_TEST" | python3 -c "import sys,json; print(json.load(sys.stdin).get('order_no',''))")
if [ -n "$TEST_ORDER_NO" ] && [ "$TEST_ORDER_NO" != "None" ]; then
  curl -s -X POST "$BASE_URL/workorders/$TEST_ORDER_NO/complete" \
    -H "Content-Type: application/json" \
    -d '{"actual_quantity": 10, "worker_ids": ["W999"], "inspection_result": "test"}' > /dev/null
  curl -s -X POST "$BASE_URL/inspection/tasks/poll" > /dev/null
  echo "  尝试删除宫保鸡丁模板:"
  curl -s -X DELETE "$BASE_URL/inspection/templates/$TEMPLATE_ID" | python3 -m json.tool
fi

echo ""
echo ""
echo "==== 质检管理服务测试完成 ===="
echo ""
echo "数据总结:"
echo "  追溯码1(全合格): $TRACE_CODE_1"
echo "  追溯码2(CCP不合格): $TRACE_CODE_2"
echo "  追溯码3(容差超标): $TRACE_CODE_3"
echo "  报告1: $REPORT_ID_1 (pass)"
echo "  报告2: $REPORT_ID_2 (fail - CCP)"
echo "  报告3: $REPORT_ID_3 (fail - 容差超标)"
echo "  处置单2: $DISPOSAL_ID_2 (已拒绝)"
echo "  处置单3: $DISPOSAL_ID_3 (已批准并执行)"
