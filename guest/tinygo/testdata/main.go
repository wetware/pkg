package main

import (
	"fmt"

	ww "github.com/wetware/ww/guest/tinygo"
)

func main() {
	// it, release := ww.Ls(context.Background())
	// defer release()

	// for name := it.Next(); name != ""; name = it.Next() {
	// 	fmt.Println(name)
	// }

	fmt.Println(ww.Test(40, 2))
}
