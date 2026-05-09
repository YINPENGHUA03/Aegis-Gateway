--KEYS[1] = "resource:stock:{id}"   库存
--KEYS[2] = "resource:buyers:{id}"  已购集合
--ARGV[1] = user_id

redis.call("INCR",KEYS[1])
redis.call("SREM",KEYS[2],ARGV[1])
return 1
