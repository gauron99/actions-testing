try:
  from . import mymath
except:
  from mymath import plus,minus,times,divide

def main():
  a = 5
  b = 2
  print(plus(a,b))
  print(minus(a,b))
  print(times(a,b))
  print(divide(a,b))

main()