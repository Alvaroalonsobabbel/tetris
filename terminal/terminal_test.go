package terminal

import (
	"reflect"
	"testing"
	"tetris/proto"
	"tetris/tetris"
)

func TestLocalStack(t *testing.T) {
	td := &templateData{
		Local: tetris.NewTestTetris(tetris.J),
	}
	want := [20][10]string{}
	for y := range want {
		for x := range want[y] {
			want[y][x] = "  "
		}
	}
	blueCell := "\x1b[7m\x1b[34m[]\x1b[0m"
	want[0][3] = blueCell
	want[1][3] = blueCell
	want[1][4] = blueCell
	want[1][5] = blueCell
	want[19][3] = "[]"
	want[18][3] = "[]"
	want[19][4] = "[]"
	want[19][5] = "[]"
	got := localStack(td)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("want %v, got %v", want, got)
	}
}

func TestRemoteStack(t *testing.T) {
	td := &templateData{
		Remote: &proto.GameMessage{
			Stack: stack2Proto(tetris.NewTestTetris(tetris.J)),
		},
	}
	want := [20][10]string{}
	for y := range want {
		for x := range want[y] {
			want[y][x] = "  "
		}
	}
	blueCell := "\x1b[7m\x1b[34m[]\x1b[0m"
	want[0][3] = blueCell
	want[1][3] = blueCell
	want[1][4] = blueCell
	want[1][5] = blueCell
	got := remoteStack(td)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("want %v, got %v", want, got)
	}
}

func TestNextPiece(t *testing.T) {
	tests := []struct {
		shape tetris.Shape
		want  []string
	}{
		{tetris.J, []string{"\x1b[7m\x1b[34m[]\x1b[0m      ", "\x1b[7m\x1b[34m[]\x1b[0m\x1b[7m\x1b[34m[]\x1b[0m\x1b[7m\x1b[34m[]\x1b[0m  "}},
		{tetris.O, []string{"\x1b[7m\x1b[33m[]\x1b[0m\x1b[7m\x1b[33m[]\x1b[0m    ", "\x1b[7m\x1b[33m[]\x1b[0m\x1b[7m\x1b[33m[]\x1b[0m    "}},
		{tetris.I, []string{"        ", "\x1b[7m\x1b[36m[]\x1b[0m\x1b[7m\x1b[36m[]\x1b[0m\x1b[7m\x1b[36m[]\x1b[0m\x1b[7m\x1b[36m[]\x1b[0m"}},
	}
	for _, tt := range tests {
		t.Run(string(tt.shape), func(t *testing.T) {
			td := &templateData{Local: tetris.NewTestTetris(tt.shape)}
			got := nextPiece(td)
			if !reflect.DeepEqual(tt.want, got) {
				t.Errorf("want %v, got %v", tt.want, got)
			}
		})
	}
}

func TestStack2Proto(t *testing.T) {
	got := stack2Proto(tetris.NewTestTetris(tetris.J))
	want := &proto.Tetris{
		Stack:      &proto.Stack{Rows: make([]*proto.Row, 20)},
		LinesClear: 0,
	}
	for i := range want.Stack.Rows {
		want.Stack.Rows[i] = &proto.Row{
			Cells: make([]string, 10),
		}
	}
	want.Stack.Rows[19].Cells[3] = "J"
	want.Stack.Rows[18].Cells[3] = "J"
	want.Stack.Rows[18].Cells[4] = "J"
	want.Stack.Rows[18].Cells[5] = "J"

	if !reflect.DeepEqual(want, got) {
		t.Errorf("want %v, got %v", want, got)
	}
}
