package carddb

import (
	"database/sql"
	"testing"
	"time"
)

const testDB = "test.db"

func init() {
	schema = `
DROP TABLE IF EXISTS deck;
DROP TABLE IF EXISTS card;
DROP TABLE IF EXISTS deck_card;
` + schema
}

func cardsEqual(c1, c2 *Card) bool {
	return c1.ID == c2.ID || c1.Front == c2.Front || c1.Back == c2.Back ||
		c1.Views == c2.Views || c1.LastView.Equal(c2.LastView)
}

func TestOpenDB(t *testing.T) {
	db, e := OpenDatabase(testDB)
	defer db.Close()
	if e != nil {
		t.Fatal(e)
	}
}

func TestNewDeck(t *testing.T) {
	db, e := OpenDatabase(testDB)
	defer db.Close()
	if e != nil {
		t.Fatal(e)
	}

	want := Deck{
		ID:         1,
		Name:       "DeckName",
		DateWeight: 1.0,
		ViewWeight: 1.0,
		ViewLimit:  1,
	}
	got, e := db.NewDeck(want.Name)
	if e != nil {
		t.Fatal(e)
	}

	if *got != want {
		t.Errorf("got: %#v want: %#v", *got, want)
	}
}

func TestUpdateDeck(t *testing.T) {
	db, e := OpenDatabase(testDB)
	defer db.Close()
	if e != nil {
		t.Fatal(e)
	}

	want := Deck{
		ID:         1,
		Name:       "DeckName",
		DateWeight: 2.0,
		ViewWeight: 3.0,
		ViewLimit:  4,
	}
	if _, e := db.NewDeck(want.Name); e != nil {
		t.Fatal(e)
	}

	if e := db.UpdateDeck(&want); e != nil {
		t.Fatal(e)
	}

	row := db.QueryRow(`SELECT deck_id, name, date_weight, view_weight, view_limit
FROM deck WHERE deck_id=?`, want.ID)
	got := Deck{}
	if e := row.Scan(&got.ID, &got.Name, &got.DateWeight, &got.ViewWeight, &got.ViewLimit); e != nil {
		t.Fatal(e)
	}

	if got != want {
		t.Errorf("got: %#v want: %#v", got, want)
	}
}

func TestDelDeck(t *testing.T) {
	db, e := OpenDatabase(testDB)
	defer db.Close()
	if e != nil {
		t.Fatal(e)
	}

	deck, e := db.NewDeck("DeckName")
	if e != nil {
		t.Fatal(e)
	}

	if e = db.DelDeck(deck.ID); e != nil {
		t.Fatal(e)
	}

	row := db.QueryRow(`SELECT deck_id FROM deck WHERE deck_id=?`, deck.ID)
	var id int
	if e = row.Scan(&id); e != sql.ErrNoRows {
		t.Errorf("got: %v want: %v", e, sql.ErrNoRows)
	}
}

func TestGetDeck(t *testing.T) {
	db, e := OpenDatabase(testDB)
	defer db.Close()
	if e != nil {
		t.Fatal(e)
	}

	deck1, e := db.NewDeck("Deck1")
	if e != nil {
		t.Fatal(e)
	}

	d := db.GetDeck(deck1.ID)
	if *d != *deck1 {
		t.Errorf("got: %#v wanted: %#v", *d, *deck1)
	}

	deck2, e := db.NewDeck("Deck2")
	if e != nil {
		t.Fatal(e)
	}

	card, e := db.NewCard()
	if e != nil {
		t.Fatal(e)
	}

	if e = db.AddCardToDeck(card.ID, deck1.ID); e != nil {
		t.Fatal(e)
	}

	ds, e := db.GetDecks(-1)
	if e != nil {
		t.Fatal(e)
	}
	if len(ds) != 2 {
		t.Errorf("got: %#v, wanted 2 decks", ds)
	}

	ds, e = db.GetDecks(0)
	if e != nil {
		t.Fatal(e)
	}
	if len(ds) != 1 && *ds[0] != *deck2 {
		t.Errorf("got: %#v, wanted: %#v", ds, *deck2)
	}

	ds, e = db.GetDecks(card.ID)
	if e != nil {
		t.Fatal(e)
	}
	if len(ds) != 1 && *ds[0] != *deck1 {
		t.Errorf("got: %#v, wanted: %#v", ds, *deck1)
	}
}

func TestNewCard(t *testing.T) {
	db, e := OpenDatabase(testDB)
	defer db.Close()
	if e != nil {
		t.Fatal(e)
	}

	want := Card{
		ID:       1,
		Front:    "NewCard",
		Back:     "",
		Views:    0,
		LastView: time.Time{},
	}
	got, e := db.NewCard()
	if e != nil {
		t.Fatal(e)
	}

	if !cardsEqual(got, &want) {
		t.Errorf("got: %#v want: %#v", *got, want)
	}
}

func TestUpdateCard(t *testing.T) {
	db, e := OpenDatabase(testDB)
	defer db.Close()
	if e != nil {
		t.Fatal(e)
	}

	want := Card{
		ID:       1,
		Front:    "Front",
		Back:     "Back",
		Views:    1,
		LastView: time.Now(),
	}
	if _, e := db.NewCard(); e != nil {
		t.Fatal(e)
	}

	if e := db.UpdateCard(&want); e != nil {
		t.Fatal(e)
	}

	row := db.QueryRow(`
SELECT card_id, front, back, views, last_view
FROM card WHERE card_id=?`, want.ID)
	got := Card{}
	if e := row.Scan(&got.ID, &got.Front, &got.Back, &got.Views, &got.LastView); e != nil {
		t.Fatal(e)
	}

	if !cardsEqual(&got, &want) {
		t.Errorf("got: %#v want: %#v", got, want)
	}
}

func TestDelCard(t *testing.T) {
	db, e := OpenDatabase(testDB)
	defer db.Close()
	if e != nil {
		t.Fatal(e)
	}

	card, e := db.NewCard()
	if e != nil {
		t.Fatal(e)
	}

	if e = db.DelCard(card.ID); e != nil {
		t.Fatal(e)
	}

	row := db.QueryRow(`SELECT card_id FROM card WHERE card_id=?`, card.ID)
	var id int
	if e = row.Scan(&id); e != sql.ErrNoRows {
		t.Errorf("got: %v want: %v", e, sql.ErrNoRows)
	}
}

func TestGetCard(t *testing.T) {
	db, e := OpenDatabase(testDB)
	defer db.Close()
	if e != nil {
		t.Fatal(e)
	}

	deck, e := db.NewDeck("Deck")
	if e != nil {
		t.Fatal(e)
	}

	card1, e := db.NewCard()
	if e != nil {
		t.Fatal(e)
	}

	c := db.GetCard(card1.ID)
	if !cardsEqual(c, card1) {
		t.Errorf("got: %#v wanted: %#v", *c, *card1)
	}

	card2, e := db.NewCard()
	if e != nil {
		t.Fatal(e)
	}

	if e = db.AddCardToDeck(card1.ID, deck.ID); e != nil {
		t.Fatal(e)
	}

	cs, e := db.GetCards(-1)
	if e != nil {
		t.Fatal(e)
	}
	if len(cs) != 2 {
		t.Errorf("got: %#v, wanted 2 decks", cs)
	}

	cs, e = db.GetCards(0)
	if e != nil {
		t.Fatal(e)
	}
	if len(cs) != 1 && *cs[0] != *card2 {
		t.Errorf("got: %#v, wanted: %#v", cs, *card2)
	}

	cs, e = db.GetCards(deck.ID)
	if e != nil {
		t.Fatal(e)
	}
	if len(cs) != 1 && !cardsEqual(cs[0], card1) {
		t.Errorf("got: %#v, wanted: %#v", cs, *card1)
	}
}

func TestAddCardToDeck(t *testing.T) {
	db, e := OpenDatabase(testDB)
	defer db.Close()
	if e != nil {
		t.Fatal(e)
	}

	deck, e := db.NewDeck("DeckName")
	if e != nil {
		t.Fatal(e)
	}

	card, e := db.NewCard()
	if e != nil {
		t.Fatal(e)
	}

	if e = db.AddCardToDeck(card.ID, deck.ID); e != nil {
		t.Fatal(e)
	}

	row := db.QueryRow(`
	SELECT card_id, deck_id
	FROM deck_card WHERE deck_id=?`, deck.ID)
	var gotCardID int
	var gotDeckID int
	if e = row.Scan(&gotCardID, &gotDeckID); e != nil {
		t.Fatal(e)
	}

	if gotCardID != card.ID {
		t.Errorf("card_id got: %d want: %d", gotCardID, card.ID)
	}

	if gotDeckID != deck.ID {
		t.Errorf("deck_id got: %d want: %d", gotDeckID, deck.ID)
	}
}

func TestDelCardToDeck(t *testing.T) {
	db, e := OpenDatabase(testDB)
	defer db.Close()
	if e != nil {
		t.Fatal(e)
	}

	deck, e := db.NewDeck("DeckName")
	if e != nil {
		t.Fatal(e)
	}

	card, e := db.NewCard()
	if e != nil {
		t.Fatal(e)
	}

	if e = db.AddCardToDeck(card.ID, deck.ID); e != nil {
		t.Fatal(e)
	}

	if e = db.DelCardFromDeck(card.ID, deck.ID); e != nil {
		t.Fatal(e)
	}

	row := db.QueryRow(`SELECT deck_id FROM deck_card WHERE deck_id=?`, deck.ID)
	var id int
	if e = row.Scan(&id); e != sql.ErrNoRows {
		t.Errorf("got: %v want: %v", e, sql.ErrNoRows)
	}
}

func TestLastView(t *testing.T) {
	db, e := OpenDatabase(testDB)
	defer db.Close()
	if e != nil {
		t.Fatal(e)
	}

	lastView, e := time.ParseInLocation("2006-1-2 15:04:05", "1234-5-6 12:34:56", time.Local)
	if e != nil {
		t.Fatal(e)
	}
	c := Card{
		ID:       1,
		Front:    "Front",
		Back:     "Back",
		Views:    1,
		LastView: lastView,
	}
	if _, e = db.NewCard(); e != nil {
		t.Fatal(e)
	}

	if e = db.UpdateCard(&c); e != nil {
		t.Fatal(e)
	}

	got := db.GetCard(c.ID)
	if !got.LastView.Equal(c.LastView) {
		t.Errorf("got: %#v wanted: %#v", got.LastView, c.LastView)
	}

	cs, e := db.GetCards(-1)
	if e != nil {
		t.Fatal(e)
	}
	got = cs[0]
	if !got.LastView.Equal(c.LastView) {
		t.Errorf("got: %#v, wanted: %#v", got.LastView, c.LastView)
	}
}
