package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/queue"
)

func LoadFile() ([]string, error) {
	Load_Codes := make([]string, 0, 100000)
	file, err := os.Open("D:/Go_projects/Golang_parser/cmd/ozon_code.txt")
	if err != nil {
		log.Println("Ошибка открытия файла:", err)
		return nil, err
	}
	defer file.Close()

	scan := bufio.NewScanner(file)

	for scan.Scan() {
		url := fmt.Sprintf("https://www.ozon.ru/search/?text=%s&from_global=true", scan.Text())
		Load_Codes = append(Load_Codes, url)
	}
	if err := scan.Err(); err != nil {
		log.Println("Ошибка чтения файла:", err)
		return nil, err
	}
	return Load_Codes, nil
}

func main() {
	data, err := LoadFile()
	if err != nil {
		log.Fatalln(err)
		return
	}
	c := colly.NewCollector(
		colly.AllowedDomains("www.ozon.ru"),
		colly.Async(true),
		colly.CacheDir("cache"),
		colly.MaxDepth(2),
		colly.AllowURLRevisit(),
	)

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*ozon.*",
		Parallelism: 2,
		RandomDelay: 5 * time.Second,
	})
	q, _ := queue.New(
		2,
		&queue.InMemoryQueueStorage{MaxSize: 100000},
	)
	c.OnError(func(r *colly.Response, err error) {
		log.Println("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})
	/*
		c.OnRequest(func(r *colly.Request) {
			log.Println("visit: ", r.URL)
		})
	*/
	c.OnHTML(".tile-hover-target", func(h *colly.HTMLElement) {
		link := h.Attr("href")
		log.Println("link: ", link)
		c.Visit(h.Request.AbsoluteURL(link))
	})

	for i := 0; i < len(data); i++ {
		q.AddURL(data[i])
	}
	if err = q.Run(c); err != nil {
		log.Fatalln(err)
	}
}
