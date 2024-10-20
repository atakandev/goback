package modules

import (
	"database/sql"
	"fmt"
	"log"
	"microservices/hooks" // hooks paketini içe aktar
	"time"
)

func updatePrice(db *sql.DB, price float64, tokenAddress string) error {
	query := `
		UPDATE public.tokens
		SET nowprice = $1
		WHERE address = $2
	`
	_, err := db.Exec(query, price, tokenAddress)
	return err
}

// autoPrice fonksiyonu
func AutoPrice(db *sql.DB) {
	startTime := time.Now()

	rows, err := db.Query("SELECT address, api FROM public.tokens;")
	if err != nil {
		log.Fatalf("Veritabanı hatası: %v", err)
	}
	defer rows.Close()

	var counter int
	const maxTokens = 1500

	for rows.Next() {
		var address, api string
		if err := rows.Scan(&address, &api); err != nil {
			log.Fatalf("Satır okuma hatası: %v", err)
		}

		if counter >= maxTokens {
			break
		}

		var tokenData *hooks.TokenData // hooks.TokenData türünü kullan
		switch api {
		case "dexscreener":
			tokenData, err = hooks.Dexscreener(address)
		case "raydium":
			tokenData, err = hooks.Raydium(address)
		case "jup.ag":
			tokenData, err = hooks.Jup(address)
		case "geckoterminal":
			tokenData, err = hooks.Geckoterminal(address)
		default:
			tokenData, err = hooks.Dexscreener(address)
		}

		if err != nil {
			continue
		}
		if tokenData != nil && tokenData.PriceUsd > 0 {
			err = updatePrice(db, tokenData.PriceUsd, address) // updatePrice fonksiyonunu uygun şekilde tanımlayın
			if err != nil {
				log.Printf("Fiyat güncelleme hatası: %v", err)
			}
		}
		counter++
	}

	// İşlem süresini hesapla
	duration := time.Since(startTime)
	fmt.Printf("İşlem süresi: %d dakika %d saniye.\n", int(duration.Minutes()), int(duration.Seconds())%60)

	// 2 dakika sonra tekrar çalıştır
	time.Sleep(2 * time.Minute)
	AutoPrice(db) // Kendini tekrar çağır
}
