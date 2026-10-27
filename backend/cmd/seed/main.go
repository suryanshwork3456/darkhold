package main

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"darkhold/internal/auth"
	"darkhold/internal/config"
	"darkhold/internal/db"
	"darkhold/internal/items"
	"darkhold/internal/wallet"
)

// All money in paise: 499900 = Rs 4,999.00
var seedItems = []items.Item{
	{ID: "item-neon-weekend", Name: "Neon Nights Weekend Pass", Category: "pass",
		Description: "3-day festival pass, general entry", FaceValue: 499900},
	{ID: "item-crimson-vip", Name: "Crimson Moon VIP Pass", Category: "pass",
		Description: "VIP lounge and front-stage access", FaceValue: 1299900},
	{ID: "item-eclipse-day1", Name: "Eclipse Fest Day 1", Category: "pass",
		Description: "Single-day entry, opening night", FaceValue: 199900},
	{ID: "item-darkhold-hoodie", Name: "Darkhold Limited Hoodie", Category: "merch",
		Description: "Numbered edition, 500 pieces", FaceValue: 299900},
	{ID: "item-raven-poster", Name: "Raven Tour Signed Poster", Category: "merch",
		Description: "Signed and framed, edition of 200", FaceValue: 149900},
}

type seedUser struct {
	email, name  string
	deposit      int64            // paise
	holdings     map[string]int64 // itemID -> qty
}

var seedUsers = []seedUser{
	{"alice@darkhold.test", "Alice", 5_000_000, nil}, // buyer, Rs 50,000
	{"bob@darkhold.test", "Bob", 1_000_000, map[string]int64{
		"item-neon-weekend": 5, "item-crimson-vip": 2, "item-darkhold-hoodie": 3}},
	{"carol@darkhold.test", "Carol", 1_000_000, map[string]int64{
		"item-eclipse-day1": 4, "item-raven-poster": 2, "item-neon-weekend": 3}},
}

const password = "password123"

func main() {
	ctx := context.Background()
	cfg := config.Load()

	database, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer database.Close()

	itemStore := items.NewStore(database)
	authStore := auth.NewStore(database)
	walletStore := wallet.NewStore(database)

	for _, it := range seedItems {
		_, err := itemStore.GetItem(ctx, it.ID)
		if err == nil {
			log.Printf("item exists, skipping: %s", it.ID)
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			log.Fatalf("check item %s: %v", it.ID, err)
		}
		if _, err := itemStore.CreateItem(ctx, it); err != nil {
			log.Fatalf("create item %s: %v", it.ID, err)
		}
		log.Printf("created item: %s", it.Name)
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		log.Fatalf("hash: %v", err)
	}

	for _, su := range seedUsers {
		if _, err := authStore.GetUserByEmail(ctx, su.email); err == nil {
			log.Printf("user exists, skipping: %s", su.email)
			continue
		} else if !errors.Is(err, sql.ErrNoRows) {
			log.Fatalf("check user %s: %v", su.email, err)
		}

		u, err := authStore.CreateUser(ctx, su.email, hash, su.name)
		if err != nil {
			log.Fatalf("create user %s: %v", su.email, err)
		}
		if err := walletStore.Deposit(ctx, u.ID, su.deposit); err != nil {
			log.Fatalf("deposit %s: %v", su.email, err)
		}
		for itemID, qty := range su.holdings {
			if err := walletStore.AddHolding(ctx, u.ID, itemID, qty); err != nil {
				log.Fatalf("holding %s/%s: %v", su.email, itemID, err)
			}
		}
		log.Printf("created user: %s", su.email)
	}

	log.Printf("seed done. login with any seeded email, password %q", password)
}