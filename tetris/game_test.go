package tetris

// import (
// 	"fmt"
// 	"sync"
// 	"testing"
// 	"time"
// )

// func TestGameOver(t *testing.T) {

// 	game := NewGame()
// 	isGameOver := make(chan bool)
// 	doneCh := make(chan bool)
// 	var wg sync.WaitGroup
// 	go func() {
// 		for {
// 			select {
// 			case u := <-game.UpdateCh:
// 				// fmt.Println(u)
// 				if u == nil {
// 					t.Error("expected update not to be nil")
// 				}
// 			case <-game.GameOverCh:
// 				fmt.Println("got to game over in go func")
// 				isGameOver <- true
// 			case <-doneCh:
// 				fmt.Println("doneCh did their thing")
// 				return
// 			}
// 		}
// 	}()

// 	go func() {
// 		select {
// 		case <-isGameOver:
// 			fmt.Println("got to game over over here")
// 			doneCh <- true
// 			wg.Done()

// 		case <-time.After(1 * time.Second):
// 			t.Error("expected to have game over but timed out")
// 			fmt.Println("this failed with timeout")
// 			doneCh <- true
// 			wg.Done()
// 		}
// 	}()

// 	wg.Add(1)
// 	game.Start()
// 	for range 14 {
// 		game.Action(DropDown)
// 		time.Sleep(10 * time.Millisecond)
// 	}
// 	wg.Wait()

// 	// i := <-isGameOver
// 	// if !i {
// 	// 	t.Errorf("wanted the game to be over, got %v", i)
// 	// }

// 	// fmt.Printf("isGameOver outside the loop\n  %v", isGameOver)
// 	// for range 15 {
// 	// 	fmt.Printf("isGameOver inside the loop  %v\n", isGameOver)
// 	// 	if isGameOver {
// 	// 		break
// 	// 	}
// 	// 	time.Sleep(15 * time.Millisecond)
// 	// 	game.Action(DropDown)
// 	// 	// if isGameOver {
// 	// 	// 	break
// 	// 	// }
// 	// }
// 	// <-game.doneCh
// 	// fmt.Println("reached here!")

// 	// select {
// 	// case <-testDoneCh:
// 	// 	// Game over was detected
// 	// case <-time.After(2 * time.Second):
// 	// 	t.Fatal("Test timed out waiting for game over")
// 	// }

// 	// if isGameOver {
// 	// 	t.Errorf("expected game to be over")
// 	// }
// }
