from .mymath import plus,minus,times,divide

def test_add():
  assert plus(1,2) == 3
  assert plus(0.5,0.5) == 1
  assert plus(1,1.5) == 2.5
  assert plus(11,-1) == 10
