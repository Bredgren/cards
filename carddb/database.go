package carddb

import (
	"database/sql"
	"time"

	// For sqlite3 driver
	_ "github.com/mattn/go-sqlite3"
)

var (
	schema = `
CREATE TABLE IF NOT EXISTS deck (
  deck_id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  date_weight FLOAT DEFAULT 1.0,
  view_weight FLOAT DEFAULT 1.0,
  -- Number of views before views no longer has an effect on the weight
  view_limit INTEGER DEFAULT 1
);

CREATE TABLE IF NOT EXISTS card (
  card_id INTEGER PRIMARY KEY AUTOINCREMENT,
  front TEXT DEFAULT 'NewCard',
  back TEXT DEFAULT '',
  views INTEGER DEFAULT 0,
  last_view DATETIME DEFAULT (DATETIME('0001-01-01 00:00:00'))
);

CREATE TABLE IF NOT EXISTS deck_card (
  deck_id INTEGER FOREIGN_KEY REFERENCES deck(deck_id),
  card_id INTEGER FOREIGN_KEY REFERENCES card(card_id),
  -- A card cannot be in the same deck more than once, though it can be in more than one deck
  UNIQUE(deck_id, card_id)
);
`
)

// Database hols the sql.DB and other relavent items
type Database struct {
	*sql.DB
}

// OpenDatabase creates and initializes a Database from the given file
func OpenDatabase(fileName string) (*Database, error) {
	db, e := sql.Open("sqlite3", fileName)
	if e != nil {
		return nil, e
	}

	_, e = db.Exec(schema)

	return &Database{db}, e
}

// Deck represents a deck of cards
type Deck struct {
	ID         int
	Name       string
	DateWeight float64
	ViewWeight float64
	ViewLimit  int
}

// Card represents a card in a deck
type Card struct {
	ID       int
	Front    string
	Back     string
	Views    int
	LastView time.Time
}

// NewDeck creates a new deck with the given name with default settings
func (db *Database) NewDeck(name string) (*Deck, error) {
	res, e := db.Exec(`INSERT INTO deck (name) VALUES (?)`, name)
	if e != nil {
		return nil, e
	}

	id, e := res.LastInsertId()
	if e != nil {
		return nil, e
	}

	row := db.QueryRow(`
SELECT deck_id, name, date_weight, view_weight, view_limit
FROM deck WHERE deck_id=?`, id)
	deck := &Deck{}
	e = row.Scan(&deck.ID, &deck.Name, &deck.DateWeight, &deck.ViewWeight, &deck.ViewLimit)
	return deck, e
}

// UpdateDeck updates the given deck in the database to match its fields
func (db *Database) UpdateDeck(deck *Deck) error {
	_, e := db.Exec(`
UPDATE deck
SET name=?, date_weight=?, view_weight=?, view_limit=?
WHERE deck_id=?`, deck.Name, deck.DateWeight, deck.ViewWeight, deck.ViewLimit, deck.ID)
	return e
}

// DelDeck deletes the deck with the given ID
func (db *Database) DelDeck(deckID int) error {
	tx, e := db.Begin()
	if e != nil {
		return e
	}

	_, e = db.Exec(`
DELETE FROM deck
WHERE deck_id=?`, deckID)
	if e != nil {
		tx.Rollback()
		return e
	}

	_, e = db.Exec(`
DELETE FROM deck_card
WHERE deck_id=?`, deckID)
	if e != nil {
		tx.Rollback()
		return e
	}

	e = tx.Commit()
	return e
}

// GetDecks returns all decks that contain the given card. cardID = 0 returns all decks
// that contain no cards. deckID < 0 returns all decks.
func (db *Database) GetDecks(cardID int) ([]*Deck, error) {
	var rows *sql.Rows
	var e error
	if cardID < 0 {
		rows, e = db.Query(`
SELECT deck_id, name, date_weight, view_weight, view_limit
FROM deck`)
	} else if cardID == 0 {
		rows, e = db.Query(`
SELECT deck_id, name, date_weight, view_weight, view_limit
FROM deck
WHERE deck_id NOT IN (
  SELECT DISTINCT deck_id
  FROM deck_card
)`)
	} else {
		rows, e = db.Query(`
SELECT deck_id, name, date_weight, view_weight, view_limit
FROM deck
NATURAL JOIN deck_card
WHERE card_id=?`, cardID)
	}
	defer rows.Close()
	var ds []*Deck
	for rows.Next() {
		d := &Deck{}
		if e = rows.Scan(&d.ID, &d.Name, &d.DateWeight, &d.ViewWeight, &d.ViewLimit); e != nil {
			return nil, e
		}
		ds = append(ds, d)
	}
	return ds, nil
}

// NewCard creates a new card with default values
func (db *Database) NewCard() (*Card, error) {
	res, e := db.Exec(`INSERT INTO card DEFAULT VALUES`)
	if e != nil {
		return nil, e
	}

	id, e := res.LastInsertId()
	if e != nil {
		return nil, e
	}

	row := db.QueryRow(`
SELECT card_id, front, back, views, last_view
FROM card WHERE card_id=?`, id)
	card := &Card{}
	e = row.Scan(&card.ID, &card.Front, &card.Back, &card.Views, &card.LastView)
	return card, e
}

// UpdateCard updates the given card in the database to match its fields
func (db *Database) UpdateCard(card *Card) error {
	_, e := db.Exec(`
UPDATE card
SET front=?, back=?, views=?, last_view=?
WHERE card_id=?`, card.Front, card.Back, card.Views, card.LastView.UTC(), card.ID)
	return e
}

// DelCard deletes the card with the given ID
func (db *Database) DelCard(cardID int) error {
	tx, e := db.Begin()
	if e != nil {
		return e
	}

	_, e = db.Exec(`
DELETE FROM card
WHERE card_id=?`, cardID)
	if e != nil {
		tx.Rollback()
		return e
	}
	_, e = db.Exec(`
DELETE FROM deck_card
WHERE card_id=?`, cardID)
	if e != nil {
		tx.Rollback()
		return e
	}

	e = tx.Commit()
	return e
}

// GetCards returns all cards in the given deck. deckID = 0 returns all cards that belong
// to no deck. deckID < 0 returns all cards.
func (db *Database) GetCards(deckID int) ([]*Card, error) {
	var rows *sql.Rows
	var e error
	if deckID < 0 {
		rows, e = db.Query(`
SELECT card_id, front, back, views, last_view
FROM card`)
	} else if deckID == 0 {
		rows, e = db.Query(`
SELECT card_id, front, back, views, last_view
FROM card
WHERE card_id NOT IN (
  SELECT DISTINCT card_id
  FROM deck_card
)`)
	} else {
		rows, e = db.Query(`
SELECT card_id, front, back, views, last_view
FROM card
NATURAL JOIN deck_card
WHERE deck_id=?`, deckID)
	}
	defer rows.Close()
	var cs []*Card
	for rows.Next() {
		c := &Card{}
		if e = rows.Scan(&c.ID, &c.Front, &c.Back, &c.Views, &c.LastView); e != nil {
			return nil, e
		}
		cs = append(cs, c)
	}
	return cs, nil
}

// AddCardToDeck adds the card with the given cardID to the deck with the given deckID
func (db *Database) AddCardToDeck(cardID, deckID int) error {
	_, e := db.Exec(`
INSERT INTO deck_card (deck_id, card_id)
VALUES (?, ?)`, deckID, cardID)
	return e
}

// DelCardFromDeck removes the card with the given cardID from the deck with the given deckID
func (db *Database) DelCardFromDeck(cardID, deckID int) error {
	_, e := db.Exec(`
DELETE FROM deck_card
WHERE deck_id=? AND card_id=?`, deckID, cardID)
	return e
}

// // GetRandomCard return a random card from the deck. The probability of selection depends
// // on the card's view count, last view time, and the decks weights for these. If the deck
// // is empty it will return nil.
// func (d *Deck) GetRandomCard() *Card {
// 	if len(d.Cards) == 0 {
// 		return nil
// 	}

// 	now := time.Now().Truncate(time.Hour)

// 	weights := make([]float64, len(d.Cards))
// 	for i, c := range d.Cards {
// 		lastView := c.LastView.Truncate(time.Hour)
// 		count := d.MaxViews - c.ViewCount
// 		if count < 0 {
// 			count = 0
// 		}
// 		weights[i] = now.Sub(lastView).Hours()*d.DateWeight + float64(count)*d.CountWeight
// 	}

// 	return d.Cards[wrand.SelectIndex(weights)]
// }

// // Update registers a view of the card by updating the last view time to be now and
// // increments the view count.
// func (c *Card) Update() {
// 	c.LastView = time.Now()
// 	c.ViewCount++
// }

// // CardsByID sorts cards by their ID
// type CardsByID []*Card
//
// // Len for sorting interface
// func (c CardsByID) Len() int {
// 	return len(c)
// }
//
// // Less for sorting interface
// func (c CardsByID) Less(i, j int) bool {
// 	return c[i].ID < c[j].ID
// }
//
// // Swap for sorting interface
// func (c CardsByID) Swap(i, j int) {
// 	c[i], c[j] = c[j], c[i]
// }
