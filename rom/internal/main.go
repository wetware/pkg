package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/wetware/pkg/cap/view"
	"github.com/wetware/pkg/cluster/routing"
	"github.com/wetware/pkg/guest/system"
)

var ctx = context.Background()

func main() {
	sess, err := system.Login(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer sess.Logout()

	fmt.Println("Hello, Wetware!")

	// it, release := sess.View().Iter(ctx, query())
	// defer release()

	// for r := it.Next(); r != nil; r = it.Next() {
	// 	render(r)
	// }

	// die(it.Err())
}

func die(err error) {
	if err == nil {
		os.Exit(0)
	}

	fmt.Fprintln(os.Stdout, err)
	os.Exit(1)
}

func query() view.Query {
	return view.NewQuery(view.All())
}

func render(r routing.Record) {
	fmt.Fprintf(os.Stdout, "/%s\n", r.Server())
}
