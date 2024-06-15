package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/debug"
	"github.com/gocolly/colly/v2/queue"
)

func LoadFile() ([]string, []string, error) {
	Load_Codes := make([]string, 0, 100000)
	Clear_Codes := make([]string, 0, 100000)
	file, err := os.Open("D:/Go_projects/Golang_parser/cmd/ozon_code.txt")
	if err != nil {
		log.Println("Ошибка открытия файла:", err)
		return nil, nil, err
	}
	defer file.Close()

	scan := bufio.NewScanner(file)
	code := ""
	for scan.Scan() {
		code = scan.Text()
		url := fmt.Sprintf("https://www.ozon.ru/search/?text=%s&from_global=true", code)
		Load_Codes = append(Load_Codes, url)
		Clear_Codes = append(Clear_Codes, code)
	}
	if err := scan.Err(); err != nil {
		log.Println("Ошибка чтения файла:", err)
		return nil, nil, err
	}

	return Load_Codes, Clear_Codes, nil
}

func main() {
	var full_categories []string = make([]string, 0, 100000)

	data, codes, err := LoadFile()
	if err != nil {
		log.Fatalln(err)
		return
	}

	outputFile, err := os.Create("ozon_results.csv")
	if err != nil {
		log.Fatalln("Ошибка создания файла:", err)
		return
	}
	defer outputFile.Close()

	writer := csv.NewWriter(outputFile)
	defer writer.Flush()
	writer.Write([]string{"Код", "Категории", "Название", "Характеристики"})

	c := colly.NewCollector(
		colly.AllowedDomains("www.ozon.ru"),
		colly.Async(true),
		colly.CacheDir("cache"),
		colly.MaxDepth(2),
		colly.AllowURLRevisit(),
		colly.Debugger(&debug.LogDebugger{}),
	)

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*ozon.*",
		Parallelism: 100000,
		RandomDelay: 1 * time.Second,
	})

	q, _ := queue.New(
		2,
		&queue.InMemoryQueueStorage{MaxSize: 100000},
	)

	c.OnRequest(func(r *colly.Request) {
		r.Ctx.Put("code", r.URL.Query().Get("text"))
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Println("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})

	c.OnHTML(".tile-hover-target", func(h *colly.HTMLElement) {
		link := h.Attr("href")
		log.Println("link: ", h.Request.AbsoluteURL(link))
		c.Visit(h.Request.AbsoluteURL(link))
	})

	c.OnHTML("ol.eg1_10", func(h *colly.HTMLElement) {
		categories := ""
		h.ForEach("li.e1g_10 > span.h5a.eg2_10", func(i int, h *colly.HTMLElement) {
			categories += h.Text + ";"
		})
		h.ForEach("li.e1g_10 > a", func(i int, h *colly.HTMLElement) {
			categories += h.Text + ";"
		})
		log.Println("Найденная категория:", categories)
		full_categories = append(full_categories, categories)
	})

	c.OnHTML("#section-characteristics", func(e *colly.HTMLElement) {
		characteristicsMap := make(map[string]string)
		e.ForEach("dl.k8p_27", func(_ int, el *colly.HTMLElement) {
			key := el.ChildText("dt.k7p_27 span.p7k_27")
			value := el.ChildText("dd.pk7_27")

			// Если значение содержит ссылку, то берём текст ссылки
			if value == "" {
				value = el.ChildText("dd.pk7_27 a")
			}

			if key != "" && value != "" {
				characteristicsMap[key] = value
			}
		})

		if len(characteristicsMap) > 0 {
			jsonData, err := json.Marshal(characteristicsMap)
			if err != nil {
				log.Println("Ошибка сериализации в JSON:", err)
				return
			}
			writer.Write([]string{e.Request.URL.String(), "", "", string(jsonData)})
		}
	})

	for i := 0; i < 20; i++ { //поменять на len(data)
		q.AddURL(data[i])
		//fmt.Println("Загруженный URL:", data[i])
	}

	if err = q.Run(c); err != nil {
		log.Panicln(err)
	}
	c.Wait()

	for i := 0; i < 20; i++ {
		writer.Write([]string{codes[i], full_categories[i]})
	}
}
