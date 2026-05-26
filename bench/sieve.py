N = 10000000

def sieve():
    s = [False] * (N+1)
    count = 0
    for i in range(2, N+1):
        if not s[i]:
            count += 1
            j = i * i
            while j <= N:
                s[j] = True
                j += i
    return count

print(sieve())
