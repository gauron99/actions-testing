package math

import "testing"

func TestAdd(t *testing.T) {
	tests := []struct {
		a   int
		b   int
		exp int
	}{
		{
			a:   10,
			b:   20,
			exp: 30,
		},
		{
			a:   1,
			b:   2,
			exp: 3,
		},
	}
	for _, tt := range tests {
		t.Run(" test int add", func(t *testing.T) {
			exp := tt.exp
			got := Plus(tt.a, tt.b)
			if exp != got {
				t.Errorf("got %v, expected %v", got, exp)
			}
		})
	}
}

func TestSub(t *testing.T) {
	tests := []struct{
		a int
		b int
		exp int
	}{
		{
			a: 10,
			b: 2,
			exp: 8,
		},
	}
	for _,tt := range tests {
		t.Run("test int subs", func(t *testing.T){
			exp:= tt.exp
			got := Minus(tt.a,tt.b)
			if exp != got {
				t.Errorf("got %v, expected %v",got,exp)
			}
		})
	}
}





			
