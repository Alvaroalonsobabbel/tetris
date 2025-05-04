package tetris

type Shape string

const (
	I Shape = "I"
	J Shape = "J"
	L Shape = "L"
	O Shape = "O"
	S Shape = "S"
	Z Shape = "Z"
	T Shape = "T"
)

var shapeMap = map[Shape]func() *Tetromino{
	I: newI,
	J: newJ,
	L: newL,
	O: newO,
	S: newS,
	Z: newZ,
	T: newT,
}

type Tetromino struct {
	Grid  [][]bool
	X     int
	Y     int
	Shape Shape
}

/*
.	Spawn Location			.	Shape

.	0 1 2 3 4 5 6 7 8 9		.	0 1 2 3

19	X X X O O O O X X X		0	X X X X

18	X X X X X X X X X X		1	O O O O

17	X X X X X X X X X X		2	X X X X

16	X X X X X X X X X X		3	X X X X
*/
func newI() *Tetromino {
	return &Tetromino{
		Grid: [][]bool{
			{false, false, false, false},
			{true, true, true, true},
			{false, false, false, false},
			{false, false, false, false},
		},
		X:     3,
		Y:     20,
		Shape: I,
	}
}

/*
.	Spawn Location		.	Shape

.	0 1 2 3 4 5 6 7 8 9		.	0 1 2

19	X X X O X X X X X X		0	O X X

18	X X X O O O X X X X		1	O O O

17	X X X X X X X X X X		2	X X X
*/
func newJ() *Tetromino {
	return &Tetromino{
		Grid: [][]bool{
			{true, false, false},
			{true, true, true},
			{false, false, false},
		},
		X:     3,
		Y:     19,
		Shape: J,
	}
}

/*
.	Spawn Location		.	Shape

.	0 1 2 3 4 5 6 7 8 9		.	0 1 2

19	X X X X X O X X X X		0	X X O

18	X X X O O O X X X X		1	O O O

17	X X X X X X X X X X		2	X X X
*/
func newL() *Tetromino {
	return &Tetromino{
		Grid: [][]bool{
			{false, false, true},
			{true, true, true},
			{false, false, false},
		},
		X:     3,
		Y:     19,
		Shape: L,
	}
}

/*
.	Spawn Location		.	Shape

.	0 1 2 3 4 5 6 7 8 9		.	0 1

19	X X X X O O X X X X		0	O O

18	X X X X O O X X X X		1	O O
*/
func newO() *Tetromino {
	return &Tetromino{
		Grid: [][]bool{
			{true, true},
			{true, true},
		},
		X:     4,
		Y:     19,
		Shape: O,
	}
}

/*
.	Spawn Location		.	Shape

.	0 1 2 3 4 5 6 7 8 9		.	0 1 2

19	X X X X O O X X X X		0	X O O

18	X X X O O X X X X X		1	O O X

17	X X X X X X X X X X		2	X X X
*/
func newS() *Tetromino {
	return &Tetromino{
		Grid: [][]bool{
			{false, true, true},
			{true, true, false},
			{false, false, false},
		},
		X:     3,
		Y:     19,
		Shape: S,
	}
}

/*
.	Spawn Location		.	Shape

.	0 1 2 3 4 5 6 7 8 9		.	0 1 2

19	X X X O O X X X X X		0	O O X

18	X X X X O O X X X X		1	X O O

17	X X X X X X X X X X		2	X X X
*/
func newZ() *Tetromino {
	return &Tetromino{
		Grid: [][]bool{
			{true, true, false},
			{false, true, true},
			{false, false, false},
		},
		X:     3,
		Y:     19,
		Shape: Z,
	}
}

/*
.	Spawn Location		.	Shape

.	0 1 2 3 4 5 6 7 8 9		.	0 1 2

19	X X X X O X X X X X		0	X O X

18	X X X O O O X X X X		1	O O O

17	X X X X X X X X X X		2	X X X
*/
func newT() *Tetromino {
	return &Tetromino{
		Grid: [][]bool{
			{false, true, false},
			{true, true, true},
			{false, false, false},
		},
		X:     3,
		Y:     19,
		Shape: T,
	}
}
