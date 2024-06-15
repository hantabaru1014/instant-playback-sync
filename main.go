package main

import "github.com/hantabaru1014/instant-playback-sync/app"

func main() {
	s := app.NewServer()
	s.Run(":8080")
}
