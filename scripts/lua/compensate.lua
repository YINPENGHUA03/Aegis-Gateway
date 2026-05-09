-- KEYS[1]: 库存  KEYS[2]: 已购集合  ARGV[1]: user_id

-- 用户不在集合里 = 已经补偿过，直接跳过
if redis.call("SISMEMBER", KEYS[2], ARGV[1]) == 0 then
    return 0
end

redis.call("INCR", KEYS[1])
redis.call("SREM", KEYS[2], ARGV[1])
return 1
