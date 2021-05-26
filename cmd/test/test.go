package main

import (
	"log"
	"net/url"
)

func main() {
	url, err := url.Parse("http://www.baidu.com/ad/dd?a")
	if err != nil {
		log.Fatalln(err)
		return
	}
	log.Println("%s", url)

}
