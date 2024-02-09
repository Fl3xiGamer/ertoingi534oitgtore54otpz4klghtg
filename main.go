package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/beito123/nbt"
	"github.com/joho/godotenv"
	"golang.org/x/net/websocket"
)

type Auction struct {
	Uuid         string `json:"uuid"`
	Start        int    `json:"start"`
	Starting_bid int    `json:"starting_bid"`
	Bin          bool   `json:"bin"`
	Item_name    string `json:"item_name"`
	Item_bytes   string `json:"item_bytes"`
}

type Response struct {
	Success      bool      `json:"success"`
	Page         int64     `json:"page"`
	TotalPages   int64     `json:"totalPages"`
	TotalAuction int64     `json:"totalAuction"`
	LastUpdated  int       `json:"lastUpdated"`
	Auctions     []Auction `json:"auctions"`
}

type ReducedResponse struct {
	TotalPages  int64 `json:"totalPages"`
	LastUpdated int   `json:"lastUpdated"`
}

var lastUpdated = 0

var minProfit int64 = 4000000
var filter string
var gui string

var lowBin = getLowBin()

var httpClient = &http.Client{
	Timeout: time.Second * 10, // Set a reasonable timeout
}

// Add a global variable for the blacklist
var blacklist = []string{"chamber", "skin", "travel"}

func getReducedPage(page int64) ReducedResponse {
	resp, err := httpClient.Get(fmt.Sprintf("https://api.hypixel.net/skyblock/auctions?page=%d", page))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	var response ReducedResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		panic(err)
	}

	return response
}

func getLowBin() map[string]float64 {
	resp, err := http.Get("http://moulberry.codes/lowestbin.json")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	var response map[string]float64
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		panic(err)
	}

	fmt.Println("Updated Low Bins!")
	return response
}

func idFromItemBytes(item_bytes string) string {
	data, _ := base64.StdEncoding.DecodeString(item_bytes)
	gz, _ := gzip.NewReader(bytes.NewBuffer(data))
	b, _ := ioutil.ReadAll(gz)
	stream, _ := nbt.FromReader(bytes.NewBuffer(b), nbt.BigEndian)
	tag, _ := stream.ReadTag()
	r := regexp.MustCompile(` id\(String\): ([^,} ]*)`)
	str, _ := tag.ToString()
	m := r.FindAllString(str, -1)
	return strings.Split(m[0], ": ")[1]
}

func update() {
	go fmt.Print("Checking for updates: ")
	page := getReducedPage(0)
	var i int64
	totalNew := 0
	deals := 0
	var wg sync.WaitGroup

	if lastUpdated < page.LastUpdated {
		go fmt.Print("\n\tUpdating Listings: ")
		start := time.Now()
		for i = 0; i < page.TotalPages; i++ {
			wg.Add(1)

			go func(i int64) {
				resp, err := http.Get(fmt.Sprintf("https://api.hypixel.net/skyblock/auctions?page=%d", i))
				if err != nil {
					panic(err)
				}
				defer resp.Body.Close()

				var response Response
				err = json.NewDecoder(resp.Body).Decode(&response)
				if err != nil {
					panic(err)
				}

				// ... (existing code)

				for i := 0; i < len(response.Auctions); i++ {
					auction := response.Auctions[i]
					if auction.Start >= lastUpdated {
						id := idFromItemBytes(auction.Item_bytes)

						// Check if the item is in the blacklist
						if isBlacklisted(auction.Item_name) {
							continue
						}

						profit := lowBin[id] - float64(auction.Starting_bid)

						// Calculate taxes
						var taxRate float64
						if lowBin[id] < 10000000 {
							taxRate = 0.01
						} else if lowBin[id] >= 10000000 && lowBin[id] < 100000000 {
							taxRate = 0.02
						} else {
							taxRate = 0.025
						}

						taxAmount := taxRate * lowBin[id]
						totalTax := 2 * taxAmount // Taxes for listing and collecting

						profitAfterTax := profit - totalTax

						// Check if profitAfterTax is negative
						if profitAfterTax >= 0 && auction.Bin && profit >= float64(minProfit) {
							if filter != "" && !strings.Contains(filter, id) {
								// Skip if filtered
							} else {
								message := fmt.Sprintf("{\"uuid\":\"/viewauction %s\",\"name\":\"%s\",\"low\":%d,\"price\":%d,\"profit\":%d,\"profitAfterTax\":%d}", auction.Uuid, auction.Item_name, int(lowBin[id]), auction.Starting_bid, int(profit), int(profitAfterTax))
								if strings.Contains(gui, id) {
									message = "gui" + message
								}
								fmt.Println(message) // Print the message to the console for debugging
								go sendMessage(message)
								deals++
							}
						}
						totalNew++
					}
				}

				defer wg.Done()
			}(i)
		}

		wg.Wait()
		end := time.Now()
		lastUpdated = page.LastUpdated
		fmt.Printf("Completed in %s | Found %d new auctions! | Found %d deals!\n", end.Sub(start), totalNew, deals)
	} else {
		fmt.Print("No Update\n")
	}
}

// isBlacklisted checks if the item contains any word from the blacklist
func isBlacklisted(itemName string) bool {
	for _, word := range blacklist {
		if strings.Contains(strings.ToLower(itemName), strings.ToLower(word)) {
			return true
		}
	}
	return false
}

func main() {
	godotenv.Load(".env")
	min := os.Getenv("MIN_PROFIT")
	filter = os.Getenv("FILTER")
	gui = os.Getenv("GUI")

	if min != "" {
		minProfit, _ = strconv.ParseInt(os.Getenv("MIN_PROFIT"), 10, 64)
	}

	lastUpdated = getReducedPage(0).LastUpdated

	http.Handle("/socket", websocket.Handler(socket))
	server := websocket.Server{
		Config:  websocket.Config{},
		Handler: socket,
	}

	go http.ListenAndServe(":8080", server)

	lowTicker := time.NewTicker(5 * time.Minute)
	go func() {
		for {
			select {
			case <-lowTicker.C:
				lowBin = getLowBin()
			}
		}
	}()

	ticker := time.NewTicker(5 * time.Second)
	func() {
		for {
			select {
			case <-ticker.C:
				update()
			}
		}
	}()
}
