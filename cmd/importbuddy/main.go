package main

import (
	"fmt"
	"log"

	"github.com/TotallyGamerJet/discdb"
)

func run() (err error) {
	output, err := discdb.Download()
	fmt.Println(output)
	return err
	//output, err := os.Create("out.txt")
	//if err != nil {
	//	return err
	//}
	//defer func(output *os.File) {
	//	err = errors.Join(err, output.Close())
	//}(output)
	//fmt.Println(discdb.WriteLogs(0, output))
	//return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}
