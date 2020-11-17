package main

const (
	calendar = "https://www.iaai.com/LiveAuctionsCalendar"
)

type Auction struct {
	date         string
	saleListLink string
}

func main() {
	/*auctions := make([]Auction, 100)
	res, err := http.Get(calendar)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		log.Fatalf("Response code equals %d", res.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	doc.Find("body").Each(func (i int, s *goquery.Selection) {
		date := s.Find("data-list data-list--block")
		link, ok := s.Find("table-cell--status a").Attr("href")
		if !ok {
			return
		}
		auctions = append(auctions, Auction{
			date: date.Contents().Text(),
			saleListLink: link,
		})
	})*/

}
