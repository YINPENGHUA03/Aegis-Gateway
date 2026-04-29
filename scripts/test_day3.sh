#!/bin/bash
URL="http://localhost:8080/api/v1"
#Run:  ./scripts/test_day3.sh

echo "=== 1. ping ==="
curl -s -o /dev/null -w "Status: %{http_code}\n" "$URL/ping"

echo "=== 2. valid reserve ==="
curl -s -o /dev/null -w "Status: %{http_code}\n" -X POST "$URL/reserve" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"user12345","resource_id":1}'

echo "=== 3. empty body ==="
curl -s -o /dev/null -w "Status: %{http_code}\n" -X POST "$URL/reserve" \
  -H "Content-Type: application/json" -d '{}'

echo "=== 4. short user_id ==="
curl -s -o /dev/null -w "Status: %{http_code}\n" -X POST "$URL/reserve" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"abc","resource_id":1}'

echo "=== 5. resource_id=0 ==="
curl -s -o /dev/null -w "Status: %{http_code}\n" -X POST "$URL/reserve" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"user12345","resource_id":0}'
