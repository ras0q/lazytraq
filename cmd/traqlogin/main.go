package main

import (
	"context"
	"os"

	"github.com/ras0q/lazytraq/internal/traqlogin"
)

func main() {
	token, err := traqlogin.GetToken(context.Background(), os.Stdout)
	if err != nil {
		panic(err)
	}

	_ = os.WriteFile("tmp/.token", []byte(token.AccessToken), 0600)
}
