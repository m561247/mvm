local N = 10000000

local function sieve()
  local s = {}
  for k = 1, N+1 do s[k] = false end
  local count = 0
  for i = 2, N do
    if not s[i] then
      count = count + 1
      local j = i * i
      while j <= N do
        s[j] = true
        j = j + i
      end
    end
  end
  return count
end

print(sieve())
