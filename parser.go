package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/chromedp/chromedp"
	"log"
	"runtime"
	"strconv"
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
	ctx, cancel := chromedp.NewContext(context.Background(), chromedp.WithLogf(log.Printf))
	return &Parser{
		db:          db,
		mainContext: ctx,
		cancel:      cancel,
		lots:        make([]Lot, 0, 3000),
	}
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
	workerCount := runtime.NumCPU()

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

func (p *Parser) getBuyNowLots() {
	count := len(p.lots)
	tasksChan := make(chan Lot, count)
	buyNowChan := make(chan Lot, count)
	workersCount := runtime.NumCPU()

	for i := 0; i < workersCount; i++ {
		go p.getBuyNow(tasksChan, buyNowChan)
	}

	for _, lot := range p.lots {
		tasksChan <- lot
	}
	p.lots = p.lots[0:0]
	close(tasksChan)

	for i := 0; i < count; i++ {
		p.lots = append(p.lots, <-buyNowChan)
		fmt.Printf("Обработано %d лотов из %d \n", i+1, count)
	}
	close(buyNowChan)
}

func (p *Parser) getBuyNow(tasks chan Lot, buyNowLots chan Lot) {
	var buyNow bool
	for lot := range tasks {
		ctx, cancel := chromedp.NewContext(p.mainContext)
		if err := chromedp.Run(ctx, chromedp.Navigate(lot.Lot)); err != nil {
			fmt.Println("Не удалось зайти на страницу результатов аукциона", lot.Lot)
		}
		js := `(() => {
		return JSON.parse(document.querySelectorAll("#ProductDetailsVM")[0].innerText)["VehicleDetailsViewModel"]["BuyNowInd"];
		})();`

		if err := chromedp.Run(ctx,
			chromedp.WaitReady(`#ProductDetailsVM`),
			chromedp.Evaluate(js, &buyNow)); err != nil {
			fmt.Println("Ошибка обработки лота ", lot.Lot)
		}

		lot.BuyNow = strconv.FormatBool(buyNow)
		buyNowLots <- lot
		fmt.Println(buyNow)
		cancel()
		chromedp.Cancel(ctx)
	}
}
