package main

import (
	"encoding/json"
	"github.com/yhat/scrape"
	"golang.org/x/net/html"
	"gopkg.in/mgo.v2"
	//	"gopkg.in/mgo.v2/bson"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	//"net/http"

	"bytes"
	"io/ioutil"
)

type Product struct {
	Site        string
	Name        string
	Link        string
	Description string
	Price       int
	Image       string
	Features    []string
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	dirname := "." + string(filepath.Separator)

	d, err := os.Open(dirname)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	defer d.Close()

	files, err := d.Readdir(-1)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	for _, file := range files {
		if file.Mode().IsRegular() {
			if filepath.Ext(file.Name()) == ".html" {
				log.Println("Reading " + file.Name())
				dat, err := ioutil.ReadFile(file.Name())
				if err != nil {
					log.Println(err)
				}
				read(dat)
			}
		}
	}
}

func read(dat []byte) {
	//resp, err := http.Get("https://www.jumia.com.ng/android-phones/")
	//	dat, err := ioutil.ReadFile("x.html")
	//	if err != nil {
	//		log.Println(err)
	//	}

	rder := bytes.NewReader(dat)

	root, err := html.Parse(rder)
	if err != nil {
		log.Println(err)

	}

	matcher := func(n *html.Node) bool {
		if n != nil {
			return scrape.Attr(n, "class") == "product-block"
		}
		return false
	}

	products := scrape.FindAll(root, matcher)

	prods := []Product{}
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)

	}
	defer session.Close()

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)

	c := session.DB("echai").C("products")
	index := mgo.Index{
		Key: []string{"$text:name"},
	}

	err = c.EnsureIndex(index)
	if err != nil {
		log.Println(err)

	}

	for _, p := range products {
		prod := Product{}

		prod.Site = "konga"

		l := func(n *html.Node) bool {
			if n != nil {
				return scrape.Attr(n, "class") == "product-block-link"
			}
			return false
		}

		link, _ := scrape.Find(p, l)

		prod.Link = scrape.Attr(link, "href")

		t := func(n *html.Node) bool {
			if n != nil {
				return scrape.Attr(n, "class") == "product-name truncate"
			}
			return false
		}

		name, _ := scrape.Find(p, t)
		prod.Name = scrape.Text(name)

		img := func(n *html.Node) bool {
			if n != nil {
				return scrape.Attr(n, "class") == "catalog-product-image"
			}
			return false
		}

		image, _ := scrape.Find(p, img)

		prod.Image = scrape.Attr(image, "src")

		pr := func(n *html.Node) bool {
			if n != nil {
				return scrape.Attr(n, "class") == "original-price original-price-bold" || scrape.Attr(n, "class") == "special-price"
			}
			return false
		}

		price, _ := scrape.Find(p, pr)
		priceno := strings.Replace(strings.Replace(strings.Replace(scrape.Text(price), ",", "", -1), "₦", "", -1), "from", "", -1)
		num, err := strconv.Atoi(priceno)
		if err != nil {
			log.Println(err)
		}

		prod.Price = num
		err = c.Insert(prod)
		if err != nil {
			log.Println(err)
		}
		prods = append(prods, prod)
	}

	log.Printf("wrote %d products to the database", len(prods))
	jsonbyte, err := json.Marshal(prods)
	if err != nil {
		log.Println(err)
	}
	ioutil.WriteFile("data.json", jsonbyte, 0777)

}
