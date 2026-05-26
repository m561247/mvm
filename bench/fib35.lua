local function fib(i)
  if i < 2 then return i end
  return fib(i-2) + fib(i-1)
end

print(fib(35))
