--KEYS[1]:"resource:stock"库存key
--KEYS[2]:"resource:buyers"已购用户集合key
--ARGV[1]:"user"操作用户ID
--取库存
--local声明为局部变量
--终端 EVAL “--下面代码” 2 resource:stock:100 resource:buyers:100 user_001
--“2”表示2个KEY
local stock = redis.call("GET",KEYS[1])
--售罄
if tonumber(stock) <=0 then
    return 0 
end
--已经买过，检查用户是否在集合里要用 SISMEMBER,EXISTS只能检查某个 key 是否存在
if redis.call("SISMEMBER",KEYS[2],ARGV[1])==1
then
    return -1
end
--成功扣减
redis.call("DECR",KEYS[1])
redis.call("SADD",KEYS[2],ARGV[1])
return 1