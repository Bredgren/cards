package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strconv"

	"github.com/Bredgren/cards/carddb"
)

var (
	port   = 8081
	static = "."
	dbFile = "cards.db"
)

var handlers = map[string]http.HandlerFunc{
	"/deck/new":     deckNewHandler,
	"/deck/edit/":   deckEditHandler,
	"/deck/delete/": deckDeleteHandler,
	"/deck/study/":  deckStudyHandler,
	"/deck/":        deckHandler,
	"/card/new/":    cardNewHandler,
	"/card/edit/":   cardEditHandler,
	"/card/delete/": cardDeleteHandler,
	"/card/":        cardHandler,
	"/":             rootHandler,
}

var (
	db *carddb.Database
)

var tmpl = template.Must(template.New("tmpl").ParseFiles(
	"./tmpl/root.tmpl",
	"./tmpl/newDeck.tmpl",
	"./tmpl/editDeck.tmpl",
	"./tmpl/delDeck.tmpl",
	"./tmpl/studyDeck.tmpl",
	"./tmpl/showDeck.tmpl",
	"./tmpl/newCard.tmpl",
	"./tmpl/editCard.tmpl",
	"./tmpl/delCard.tmpl",
	"./tmpl/showCard.tmpl",
))

func main() {
	flag.IntVar(&port, "port", port, "HTTP port")
	flag.StringVar(&static, "s", static, "Static file directory")
	flag.StringVar(&dbFile, "db", dbFile, "SQL card database file")
	flag.Parse()

	for path, handler := range handlers {
		http.HandleFunc(path, handler)
	}

	http.Handle("/static/", http.FileServer(http.Dir(static)))

	var e error
	db, e = carddb.OpenDatabase(dbFile)
	if e != nil {
		log.Fatal(e)
	}

	addr := fmt.Sprintf(":%d", port)
	log.Println("Server started at", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	// Show list of decks with option to create/edit/delete
	rootInfo := struct {
		Decks    []*carddb.Deck
		NumCards []int
	}{}

	var e error
	rootInfo.Decks, e = db.GetDecks(-1)
	if e != nil {
		internalError(w, e)
		return
	}
	sort.Sort(carddb.DecksByName(rootInfo.Decks))

	rootInfo.NumCards = make([]int, len(rootInfo.Decks))
	for i, deck := range rootInfo.Decks {
		cards, e := db.GetCards(deck.ID)
		if e != nil {
			internalError(w, e)
			return
		}
		rootInfo.NumCards[i] = len(cards)
	}

	if e := tmpl.ExecuteTemplate(w, "Root", rootInfo); e != nil {
		internalError(w, e)
		return
	}
}

func deckNewHandler(w http.ResponseWriter, r *http.Request) {
	// Show form for creating a new deck
	if r.Method == http.MethodPost {
		if e := r.ParseForm(); e != nil {
			internalError(w, e)
			return
		}
		name := r.PostForm["name"][0]
		dateWeight, e := strconv.ParseFloat(r.PostForm["dateWeight"][0], 64)
		if e != nil {
			internalError(w, e)
			return
		}
		viewWeight, e := strconv.ParseFloat(r.PostForm["viewWeight"][0], 64)
		if e != nil {
			internalError(w, e)
			return
		}
		viewLimit, e := strconv.Atoi(r.PostForm["viewLimit"][0])
		if e != nil {
			internalError(w, e)
			return
		}

		deck, e := db.NewDeck(name)
		if e != nil {
			log.Println(e)
			if e = tmpl.ExecuteTemplate(w, "NewDeckFail", struct {
				Name  string
				Error string
			}{name, e.Error()}); e != nil {
				internalError(w, e)
				return
			}
			return
		}

		deck.DateWeight = dateWeight
		deck.ViewWeight = viewWeight
		deck.ViewLimit = viewLimit
		if e := db.UpdateDeck(deck); e != nil {
			internalError(w, e)
			return
		}

		if e := tmpl.ExecuteTemplate(w, "NewDeckSuccess", struct {
			Deck *carddb.Deck
		}{deck}); e != nil {
			internalError(w, e)
			return
		}

		return
	}

	if e := tmpl.ExecuteTemplate(w, "NewDeck", nil); e != nil {
		internalError(w, e)
		return
	}
}

func deckEditHandler(w http.ResponseWriter, r *http.Request) {
	// Show form for editing existing deck
	form, e := parseForm(r)
	if e != nil {
		log.Println(e)
		http.NotFound(w, r)
		return
	}

	if r.Method == http.MethodPost {
		name := r.PostForm["name"][0]
		dateWeight, e := strconv.ParseFloat(r.PostForm["dateWeight"][0], 64)
		if e != nil {
			internalError(w, e)
			return
		}
		viewWeight, e := strconv.ParseFloat(r.PostForm["viewWeight"][0], 64)
		if e != nil {
			internalError(w, e)
			return
		}
		viewLimit, e := strconv.Atoi(r.PostForm["viewLimit"][0])
		if e != nil {
			internalError(w, e)
			return
		}

		form.Deck.Name = name
		form.Deck.DateWeight = dateWeight
		form.Deck.ViewWeight = viewWeight
		form.Deck.ViewLimit = viewLimit
		db.UpdateDeck(form.Deck)

		if e := tmpl.ExecuteTemplate(w, "EditDeckSuccess", struct {
			Deck *carddb.Deck
		}{form.Deck}); e != nil {
			internalError(w, e)
			return
		}

		return
	}

	if e := tmpl.ExecuteTemplate(w, "EditDeck", struct {
		Deck *carddb.Deck
	}{form.Deck}); e != nil {
		internalError(w, e)
		return
	}
}

func deckDeleteHandler(w http.ResponseWriter, r *http.Request) {
	// Show confirmation page for deleting deck
	form, e := parseForm(r)
	if e != nil {
		log.Println(e)
		http.NotFound(w, r)
		return
	}

	if r.Method == http.MethodPost {
		e := db.DelDeck(form.Deck.ID)
		if e != nil {
			internalError(w, e)
			return
		}

		if e := tmpl.ExecuteTemplate(w, "DelDeckSuccess", struct {
			Deck *carddb.Deck
		}{form.Deck}); e != nil {
			internalError(w, e)
			return
		}

		return
	}

	if e := tmpl.ExecuteTemplate(w, "DelDeck", struct {
		Deck *carddb.Deck
	}{form.Deck}); e != nil {
		internalError(w, e)
		return
	}
}

func deckStudyHandler(w http.ResponseWriter, r *http.Request) {
	form, e := parseForm(r)
	if e != nil || form.Deck == nil {
		if e != nil {
			log.Println(e)
		}
		http.NotFound(w, r)
		return
	}

	if form.Card == nil {
		cards, e := db.GetCards(form.Deck.ID)
		if e != nil {
			internalError(w, e)
			return
		}
		randCard := carddb.RandomCard(form.Deck, cards)
		db.ViewCard(randCard)
		http.Redirect(w, r, fmt.Sprintf("/deck/study/?d=%d&c=%d", form.Deck.ID, randCard.ID), http.StatusFound)
		return
	}

	if form.DV != 0 {
		form.Card.Views += form.DV
		db.UpdateCard(form.Card)
		http.Redirect(w, r, fmt.Sprintf("/deck/study/?d=%d&c=%d", form.Deck.ID, form.Card.ID), http.StatusFound)
		return
	}

	if e := tmpl.ExecuteTemplate(w, "Study", struct {
		Deck *carddb.Deck
		Card *carddb.Card
	}{form.Deck, form.Card}); e != nil {
		internalError(w, e)
		return
	}
}

func deckHandler(w http.ResponseWriter, r *http.Request) {
	// Show settings and cards for a particular deck. If unspecified, redirect to root.
	form, e := parseForm(r)
	if e != nil {
		log.Println(e)
		http.NotFound(w, r)
		return
	}

	if form.Deck == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	deck := db.GetDeck(form.Deck.ID)
	cards, e := db.GetCards(deck.ID)
	if e != nil {
		internalError(w, e)
		return
	}
	sort.Sort(carddb.CardsByID(cards))
	// LastViewed: card.LastView.Format("Mon Jan 2 15:04:05 2006"),

	if e := tmpl.ExecuteTemplate(w, "ShowDeck", struct {
		Deck  *carddb.Deck
		Cards []*carddb.Card
	}{deck, cards}); e != nil {
		internalError(w, e)
		return
	}
}

func cardNewHandler(w http.ResponseWriter, r *http.Request) {
	form, e := parseForm(r)
	if e != nil {
		log.Println(e)
		http.NotFound(w, r)
		return
	}

	if r.Method == http.MethodPost {
		front := r.PostForm["front"][0]
		back := r.PostForm["back"][0]

		card, e := db.NewCard()
		if e != nil {
			internalError(w, e)
			return
		}

		card.Front = front
		card.Back = back
		if e := db.UpdateCard(card); e != nil {
			internalError(w, e)
			return
		}

		if form.Deck != nil {
			if e := db.AddCardToDeck(card.ID, form.Deck.ID); e != nil {
				internalError(w, e)
				return
			}
		}

		if e := tmpl.ExecuteTemplate(w, "NewCardSuccess", struct {
			Deck *carddb.Deck
			Card *carddb.Card
		}{form.Deck, card}); e != nil {
			internalError(w, e)
			return
		}

		return
	}

	if e := tmpl.ExecuteTemplate(w, "NewCard", struct {
		Deck *carddb.Deck
	}{form.Deck}); e != nil {
		internalError(w, e)
		return
	}
}

func cardEditHandler(w http.ResponseWriter, r *http.Request) {
	form, e := parseForm(r)
	if e != nil || form.Card == nil {
		if e != nil {
			log.Println(e)
		} else {
			log.Printf("No card with ID %s\n", r.FormValue("c"))
		}
		http.NotFound(w, r)
		return
	}

	if r.Method == http.MethodPost {
		front := r.PostForm["front"][0]
		back := r.PostForm["back"][0]
		views, e := strconv.Atoi(r.PostForm["views"][0])
		if e != nil {
			internalError(w, e)
			return
		}

		form.Card.Front = front
		form.Card.Back = back
		form.Card.Views = views

		if e := db.UpdateCard(form.Card); e != nil {
			internalError(w, e)
			return
		}

		if e := tmpl.ExecuteTemplate(w, "EditCardSuccess", struct {
			Card *carddb.Card
		}{form.Card}); e != nil {
			internalError(w, e)
			return
		}

		return
	}

	if e := tmpl.ExecuteTemplate(w, "EditCard", struct {
		Card *carddb.Card
	}{form.Card}); e != nil {
		internalError(w, e)
		return
	}
}

func cardDeleteHandler(w http.ResponseWriter, r *http.Request) {
	form, e := parseForm(r)
	if e != nil || form.Card == nil {
		log.Println(e)
		http.NotFound(w, r)
		return
	}

	if r.Method == http.MethodPost {
		if e := db.DelCard(form.Card.ID); e != nil {
			internalError(w, e)
			return
		}

		if e := tmpl.ExecuteTemplate(w, "DelCardSuccess", struct {
			Card *carddb.Card
		}{form.Card}); e != nil {
			internalError(w, e)
			return
		}
		return
	}

	if e := tmpl.ExecuteTemplate(w, "DelCard", struct {
		Card *carddb.Card
	}{form.Card}); e != nil {
		internalError(w, e)
		return
	}
}

func cardHandler(w http.ResponseWriter, r *http.Request) {
	cards, e := db.GetCards(-1)
	if e != nil {
		internalError(w, e)
		return
	}
	sort.Sort(carddb.CardsByID(cards))
	// LastViewed: card.LastView.Format("Mon Jan 2 15:04:05 2006"),

	if e := tmpl.ExecuteTemplate(w, "ShowCard", struct {
		Cards []*carddb.Card
	}{cards}); e != nil {
		internalError(w, e)
		return
	}
}

func internalError(w http.ResponseWriter, e error) {
	log.Println(e)
	http.Error(w, "Internal Error", http.StatusInternalServerError)
}

type form struct {
	Deck *carddb.Deck
	Card *carddb.Card
	DV   int
}

func parseForm(r *http.Request) (form, error) {
	f := form{}
	if e := r.ParseForm(); e != nil {
		return f, e
	}
	deckID, e := strconv.Atoi(r.FormValue("d"))
	if e != nil {
		f.Deck = nil
	} else {
		f.Deck = db.GetDeck(deckID)
		if f.Deck == nil {
			return f, fmt.Errorf("No deck with ID %d", deckID)
		}
	}

	cardID, e := strconv.Atoi(r.FormValue("c"))
	if e != nil {
		f.Card = nil
	} else {
		f.Card = db.GetCard(cardID)
		if f.Card == nil {
			return f, fmt.Errorf("No card with ID %d", cardID)
		}
	}

	dvStr := r.FormValue("dv")
	if dvStr != "" {
		dv, e := strconv.Atoi(dvStr)
		if e != nil {
			return f, e
		}
		f.DV = dv
	}

	return f, nil
}
