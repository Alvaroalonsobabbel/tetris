package client

import (
	"log/slog"
	"os"
	"reflect"
	"strings"
	"testing"
	"tetris/proto"
	"tetris/tetris"

	approvals "github.com/approvals/go-approval-tests"
)

func TestRender(t *testing.T) {
	tests := []struct {
		name       string
		renderFunc func(*render)
	}{
		{
			name:       "lobby with no data",
			renderFunc: func(r *render) { r.lobby() },
		},
		{
			name:       "local receiving tetris.T",
			renderFunc: func(r *render) { r.local(tetris.NewTestTetris(tetris.T)) },
		},
		{
			name: "local receiving gameOver renders lobby",
			renderFunc: func(r *render) {
				tts := tetris.NewTestTetris(tetris.T)
				tts.GameOver = true
				r.local(tts)
			},
		},
	}
	tmp, err := loadTemplate()
	if err != nil {
		t.Fatalf("failed to load template: %v", err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &strings.Builder{}
			r := &render{
				writer:       w,
				logger:       slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})),
				template:     tmp,
				templateData: &templateData{},
			}
			tt.renderFunc(r)
			approvals.VerifyString(t, w.String())
		})
	}
}

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

	t.Run("localStack with nil tetris returns emtpy spaces", func(t *testing.T) {
		want := [20][10]string{}
		for y := range 20 {
			for x := range 10 {
				want[y][x] = "  "
			}
		}
		got := localStack(nil)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("want %v, got %v", want, got)
		}
	})
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

	t.Run("remoteStack with nil tetris returns emtpy spaces", func(t *testing.T) {
		want := [20][10]string{}
		for y := range 20 {
			for x := range 10 {
				want[y][x] = "  "
			}
		}
		got := remoteStack(nil)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("want %v, got %v", want, got)
		}
	})
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
	t.Run("nextPiece with nil tetris returns emtpy spaces", func(t *testing.T) {
		want := []string{"        ", "        "}
		got := nextPiece(nil)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("want %v, got %v", want, got)
		}
	})
}

func TestStack2Proto(t *testing.T) {
	got := stack2Proto(tetris.NewTestTetris(tetris.J))
	want := &proto.Stack{Rows: make([]*proto.Row, 20)}

	for i := range want.Rows {
		want.Rows[i] = &proto.Row{
			Cells: make([]string, 10),
		}
	}
	want.Rows[19].Cells[3] = "J"
	want.Rows[18].Cells[3] = "J"
	want.Rows[18].Cells[4] = "J"
	want.Rows[18].Cells[5] = "J"

	if !reflect.DeepEqual(want, got) {
		t.Errorf("want %v, got %v", want, got)
	}
}
