package modules

import (
	"database/sql"
	"log"
	"microservices/hooks"
)

func updateNullToken(db *sql.DB, price float64, api string, tokenAddress string) error {
	query := `
		UPDATE public.tokens
		SET nowprice = $1, api = $2
		WHERE address = $3
	`
	_, err := db.Exec(query, price, api, tokenAddress)
	return err
}

func NullControl(db *sql.DB) {
	nullPriceRows, err := db.Query("SELECT address FROM public.tokens WHERE nowprice IS NULL;") // Sadece address al
	if err != nil {
		log.Fatalf("Veritabanı hatası: %v", err)
	}
	defer nullPriceRows.Close()

	for nullPriceRows.Next() {
		var address string
		if err := nullPriceRows.Scan(&address); err != nil {
			log.Fatalf("Satır okuma hatası: %v", err)
		}

		var tokenData *hooks.TokenData
		tokenData, err = hooks.Dexscreener(address) //dexscreener
		if err == nil && tokenData != nil && tokenData.PriceUsd > 0 {
			log.Printf("Güncellenen Token (dexscreener): %s, Yeni Fiyat: $%.2f", tokenData.TokenName, tokenData.PriceUsd)

			err = updateNullToken(db, tokenData.PriceUsd, "dexscreener", address)
			if err != nil {
				log.Printf("Fiyat güncelleme hatası: %v", err)
			}
			continue
		}
		tokenData, err = hooks.Jup(address) // Jup
		if err == nil && tokenData != nil && tokenData.PriceUsd > 0 {
			log.Printf("Güncellenen Token (jup.ag): %s, Yeni Fiyat: $%.2f", tokenData.TokenName, tokenData.PriceUsd)

			err = updateNullToken(db, tokenData.PriceUsd, "jup.ag", address)
			if err != nil {
				log.Printf("Fiyat güncelleme hatası: %v", err)
			}
			continue
		}
		tokenData, err = hooks.Raydium(address) // raydium
		if err == nil && tokenData != nil && tokenData.PriceUsd > 0 {
			log.Printf("Güncellenen Token (raydium): %s, Yeni Fiyat: $%.2f", tokenData.TokenName, tokenData.PriceUsd)

			err = updateNullToken(db, tokenData.PriceUsd, "raydium", address)
			if err != nil {
				log.Printf("Fiyat güncelleme hatası: %v", err)
			}
			continue
		}

		tokenData, err = hooks.Geckoterminal(address) //geckoterminal
		if err == nil && tokenData != nil && tokenData.PriceUsd > 0 {
			log.Printf("Güncellenen Token (geckoterminal): %s, Yeni Fiyat: $%.2f", tokenData.TokenName, tokenData.PriceUsd)

			err = updateNullToken(db, tokenData.PriceUsd, "geckoterminal", address)
			if err != nil {
				log.Printf("Fiyat güncelleme hatası: %v", err)
			}
			continue
		}

		//log.Printf("Fiyat bulunamadı: %s", address)
	}
	log.Printf("Bitti.")
}
