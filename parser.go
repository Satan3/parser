package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"log"
	"net/http"
	"regexp"
	"runtime"
	"sync"
)

const (
	calendar = "https://www.iaai.com/LiveAuctionsCalendar"
)

type Auction struct {
	Time string `json:"time"`
	Link string `json:"link"`
}

type Lot struct {
	Lot    string `json:"lot"`
	Year   string `json:"year"`
	Vin    string `json:"vin"`
	BuyNow string `json:"buyNow"`
}

type Parser struct {
	db          *sql.DB
	mainContext context.Context
	cancel      context.CancelFunc
	auctions    []Auction
	lots        []Lot
}

func NewParser(db *sql.DB) *Parser {
	return &Parser{
		db:   db,
		lots: make([]Lot, 0, 3000),
	}
}

func (p *Parser) initMainContext() {
	ctx, cancel := chromedp.NewContext(
		context.Background(),
		chromedp.WithLogf(log.Printf),
	)
	p.mainContext = ctx
	p.cancel = cancel
}

func (p *Parser) parse() {
	p.initMainContext()
	p.getAuctions()
	p.getAllLots()
	p.clearOldLots()
	p.insertLots()
}

func (p *Parser) actualizeBuyNow(config *Config) {
	p.getLotsFromDB()
	if len(p.lots) == 0 {
		fmt.Println("Отсутствуют лоты для проверки")
		return
	}
	p.getBuyNowLots(config.GoroutinesMultiplier)
	if config.SendTo == "telegram" {
		p.sendToTelegram(config.TelegramBotKey)
		return
	}
	p.clearOldLots()
	p.insertLots()
}

func (p *Parser) getAuctions() {
	fmt.Println("Начато получение Аукционов")
	js := `(() => {
    	let values = [];
    	document.querySelectorAll(".table-row-inner").forEach(item => {
    	    let cells = item.querySelectorAll(".table-cell");
    	    let time  = cells[1].querySelector("li");
    	    let link = cells[4].querySelector("a");
    	    time && link && values.push({time: time.textContent, link: link.href});
    	});
    	return values;
	})();`

	if err := chromedp.Run(p.mainContext, chromedp.Navigate(calendar)); err != nil {
		log.Fatalf("Не удалось зайти на сайт: %v", err)
	}

	if err := chromedp.Run(p.mainContext, chromedp.WaitReady("#dvListLiveAuctions")); err != nil {
		log.Fatalf("Не загрузился раздел с аукционами: %v", err)
	}

	if err := chromedp.Run(p.mainContext, chromedp.Evaluate(js, &p.auctions)); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Получение аукционов завершено")
}

func (p *Parser) getAllLots() {
	count := len(p.auctions)
	auctionsChan := make(chan Auction, count)
	lotsChan := make(chan []Lot, count)
	workerCount := runtime.NumCPU() * 3

	for i := 0; i < workerCount; i++ {
		go p.getLots(auctionsChan, lotsChan)
	}

	for _, auction := range p.auctions {
		auctionsChan <- auction
	}

	close(auctionsChan)

	for i := 0; i < count; i++ {
		parsedLots := <-lotsChan
		p.lots = append(p.lots, parsedLots...)
		fmt.Printf("Обработано %d аукционов из %d \n", i+1, count)
	}
	close(lotsChan)
}

func (p *Parser) getLots(tasks chan Auction, lotsChan chan []Lot) {
	lots := make([]Lot, 0, 500)
	ctx, cancel := chromedp.NewContext(p.mainContext)
	defer cancel()

	for auction := range tasks {
		if err := chromedp.Run(ctx, chromedp.Navigate(auction.Link)); err != nil {
			fmt.Println("Не удалось зайти на страницу результатов аукциона", auction.Link)
		}

		js := `(() => {
    	let values = [];
    	document.querySelectorAll("tr").forEach(item => {
    	    let cells = item.querySelectorAll("td");
    	    if (!cells.length) {
    	        return;
    	    }
    	    let lot = cells[3];
    	    if (lot) {
    	        lot = lot.querySelector("a").href;
    	    }
    	    let year = cells[6];
    	    if (year) {
    	        year = year.textContent.trim();
    	    }
    	    if (year < 2010) {
    	        return;
    	    }
    	    let vin = cells[11];
    	    if (vin) {
                vin = vin.querySelector("a")
    	        if (vin) {
                    vin = vin.textContent;
                }
    	    }
    	    values.push({lot: lot, year: year, vin: vin});
    	});
    	return values;
	})();`

		if err := chromedp.Run(ctx, chromedp.Evaluate(js, &lots)); err != nil {
			fmt.Println("Не удалось получить данные о лотах", auction.Link)
		}
		lotsChan <- lots
	}
}

func (p *Parser) getBuyNowLots(goroutinesMultiplier int) {
	wg := sync.WaitGroup{}
	count := len(p.lots)
	fmt.Println("Лотов для проверки ", count)
	tasksChan := make(chan Lot, count)
	buyNowChan := make(chan Lot, count)
	workersCount := runtime.NumCPU() * goroutinesMultiplier
	for i := 0; i < workersCount; i++ {
		wg.Add(1)
		go p.getBuyNow(tasksChan, buyNowChan, &wg)
	}

	for _, lot := range p.lots {
		tasksChan <- lot
	}
	p.lots = p.lots[0:0]
	close(tasksChan)

	go func() {
		wg.Wait()
		close(buyNowChan)
	}()

	var current int
	for lot := range buyNowChan {
		p.lots = append(p.lots, lot)
		current++
		fmt.Printf("Обработано %d лотов \n", current)
	}
}

func (p *Parser) getBuyNow(tasks chan Lot, buyNowLots chan Lot, wg *sync.WaitGroup) {
	for lot := range tasks {
		res, err := http.Get(lot.Lot)
		if err != nil {
			fmt.Println(err)
			continue
		}
		if res.StatusCode != 200 {
			fmt.Printf("status code error: %d %s \n", res.StatusCode, res.Status)
		}

		doc, err := goquery.NewDocumentFromReader(res.Body)
		if err != nil {
			fmt.Println(err)
			continue
		}

		selection := doc.Find("#ProductDetailsVM")
		regExp := regexp.MustCompile(`"BuyNowInd":(\w+),`)
		result := regExp.FindStringSubmatch(selection.Text())

		if len(result) == 2 && result[1] == "true" {
			lot.BuyNow = result[1]
			buyNowLots <- lot
		}
		res.Body.Close()
	}
	wg.Done()
}

func (p *Parser) sendToTelegram(telegramBotKey string) {
	fmt.Println(telegramBotKey)
}
