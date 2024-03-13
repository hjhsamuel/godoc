package main

import (
	"fmt"
	"godoc/parse"
)

func main() {
	//var (
	//	projPath = "D:/workspace/code/moresec/ms/ms_install"
	//	relPath  = "internal/install/config.go"
	//	name     = "Config"
	//)
	var (
		projPath = "D:/workspace/code/moresec/ms/ms_vss"
		relPath  = "internal/models/test.go"
		name     = "TestE"
	)

	g, err := parse.NewGoParser(projPath)
	if err != nil {
		panic(err)
	}
	out, err := g.Parse(relPath, name)
	if err != nil {
		panic(err)
	}
	for _, c := range out {
		fmt.Println(c.Name, c.Type)
		for _, field := range c.Fields {
			fmt.Printf("%#v\n", field)
		}
		fmt.Println("======================>")
	}
}
