package modules

import (
	"database/sql"
	"fmt"
	"log"
	"microservices/hooks" // hooks paketini içe aktar
	"time"
)

func updatePrice(db *sql.DB, price float64, tokenAddress string) error {
	// Güncel zamanı alıyoruz
	currentTime := time.Now().Format(time.RFC3339)

	// Sorgu: nowprice değerini güncelle ve history sütununa yeni fiyat ve tarih ekle
	query := `
		UPDATE public.tokens
		SET nowprice = $1,
		    history = COALESCE(
		        history || jsonb_build_array(jsonb_build_object('price', to_json($2::numeric), 'date', to_json($3::text))),
		        jsonb_build_array(jsonb_build_object('price', to_json($2::numeric), 'date', to_json($3::text)))
		    )
		WHERE address = $4
	`

	// Sorguyu çalıştır
	_, err := db.Exec(query, price, price, currentTime, tokenAddress)
	return err
}

// autoPrice fonksiyonu
func AutoPrice(db *sql.DB) {
	startTime := time.Now()

	rows, err := db.Query("SELECT address, api, tokenname FROM public.tokens;")
	if err != nil {
		log.Fatalf("Veritabanı hatası: %v", err)
	}
	defer rows.Close()

	var counter int
	const maxTokens = 4500

	for rows.Next() {
		var address, api, tokenname string
		if err := rows.Scan(&address, &api, &tokenname); err != nil {
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
		case "solscan":
			tokenData, err = hooks.SolscanMeta(address)
		default:
			tokenData, err = hooks.SolscanMeta(address)
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
	fmt.Printf("Token Fiyat İşlem süresi: %d dakika %d saniye.\n", int(duration.Minutes()), int(duration.Seconds())%60)
}
