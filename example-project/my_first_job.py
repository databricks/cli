import sys

print("hello world!")
print(sys.argv)

for x in spark.range(10).collect():
    print(x)
