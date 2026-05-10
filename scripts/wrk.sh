#!/bin/bash
# 用途：生成 wrk 压测的 lua 脚本到 /tmp/reserve.lua
# 跑法：bash scripts/wrk.sh && wrk -t8 -c200 -d30s -s /tmp/reserve.lua http://localhost:8080/api/v1/reserve

cat > /tmp/reserve.lua << 'EOF'
counter = 0

function request()
    counter = counter + 1
    local body = string.format('{"user_id":"user_%010d","resource_id":1}', counter)
    return wrk.format("POST", nil, {["Content-Type"]="application/json"}, body)
end

-- 统计每种状态码出现次数
status_codes = {}

function response(status, headers, body)
    status_codes[status] = (status_codes[status] or 0) + 1
end

function done(summary, latency, requests)
    for code, count in pairs(status_codes) do
        io.write(string.format("Status %d: %d\n", code, count))
    end
end
EOF
