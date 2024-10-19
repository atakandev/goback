package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	_ "github.com/lib/pq"
)

type TokenData struct {
	TokenName string  `json:"name"`
	Symbol    string  `json:"symbol"`
	Logo      *string `json:"imageUrl,omitempty"`
	PriceUsd  float64 `json:"priceUsd"`
	API       string  `json:"api"`
}

// Fiyat güncelleme fonksiyonu
func updatePrice(db *sql.DB, price float64, tokenAddress string) error {
	query := `
		UPDATE public.tokens
		SET nowprice = $1
		WHERE address = $2
	`
	_, err := db.Exec(query, price, tokenAddress)
	return err
}
func updateNullToken(db *sql.DB, price float64, api string, tokenAddress string) error {
	query := `
		UPDATE public.tokens
		SET nowprice = $1, api = $2
		WHERE address = $3
	`
	_, err := db.Exec(query, price, api, tokenAddress)
	return err
}

// Dış API'den fiyat verisi çeken fonksiyon
func dexscreener(mintAddress string) (*TokenData, error) {
	url := fmt.Sprintf("https://api.dexscreener.com/latest/dex/tokens/%s", mintAddress)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if pairs, ok := result["pairs"].([]interface{}); ok && len(pairs) > 0 {
		pair := pairs[0].(map[string]interface{})

		if baseToken, ok := pair["baseToken"].(map[string]interface{}); ok {
			// Token adını ve sembolünü al
			tokenName := baseToken["name"].(string)
			symbol := baseToken["symbol"].(string)

			// Logoyu kontrol et
			var logo *string
			if info, ok := pair["info"].(map[string]interface{}); ok {
				if imageUrl, ok := info["imageUrl"].(string); ok {
					logo = &imageUrl
				}
			}

			// Fiyatı al
			priceUsd, ok := pair["priceUsd"].(string)
			if !ok {
				return nil, fmt.Errorf("")
			}
			price, err := strconv.ParseFloat(priceUsd, 64)
			if err != nil {
				return nil, err
			}

			return &TokenData{
				TokenName: tokenName,
				Symbol:    symbol,
				Logo:      logo,
				PriceUsd:  price,
				API:       "dexscreener",
			}, nil
		}
	}

	return nil, fmt.Errorf("price not found")
}
func raydium(mintAddress string) (*TokenData, error) {
	try := func() (*TokenData, error) {
		// İlk API isteği
		tokenDataResponse, err := http.Get(fmt.Sprintf("https://api-v3.raydium.io/mint/ids?mints=%s", mintAddress))
		if err != nil {
			return nil, err
		}
		defer tokenDataResponse.Body.Close()

		var tokenData struct {
			Data []struct {
				Name    string  `json:"name"`
				Symbol  string  `json:"symbol"`
				LogoURI *string `json:"logoURI"`
			} `json:"data"`
		}
		if err := json.NewDecoder(tokenDataResponse.Body).Decode(&tokenData); err != nil {
			return nil, err
		}

		// Token verisini kontrol et
		if len(tokenData.Data) > 0 {
			data := tokenData.Data[0]
			// İkinci API isteği
			priceResponse, err := http.Get(fmt.Sprintf("https://api-v3.raydium.io/mint/price?mints=%s", mintAddress))
			if err != nil {
				return nil, err
			}
			defer priceResponse.Body.Close()

			var priceData struct {
				Data map[string]float64 `json:"data"`
			}
			if err := json.NewDecoder(priceResponse.Body).Decode(&priceData); err != nil {
				return nil, err
			}

			// Fiyatı al
			price, exists := priceData.Data[mintAddress]
			if !exists {
				return nil, fmt.Errorf("price not found for mint address: %s", mintAddress)
			}

			return &TokenData{
				TokenName: data.Name,
				Symbol:    data.Symbol,
				Logo:      data.LogoURI,
				PriceUsd:  price,
				API:       "raydium",
			}, nil
		}

		return nil, fmt.Errorf("no token data found for mint address: %s", mintAddress)
	}

	return try()
}
func jup(mintAddress string) (*TokenData, error) {
	resp, err := http.Get(fmt.Sprintf("https://api.jup.ag/tokens/v1/%s", mintAddress))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tokenData struct {
		Name    string  `json:"name"`
		Symbol  string  `json:"symbol"`
		LogoURI *string `json:"logoURI"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenData); err != nil {
		return nil, err
	}

	priceResp, err := http.Get(fmt.Sprintf("https://price.jup.ag/v6/price?ids=%s", mintAddress))
	if err != nil {
		return nil, err
	}
	defer priceResp.Body.Close()

	var priceData struct {
		Data map[string]struct {
			Price float64 `json:"price"`
		} `json:"data"`
	}
	if err := json.NewDecoder(priceResp.Body).Decode(&priceData); err != nil {
		return nil, err
	}

	price := priceData.Data[mintAddress].Price

	return &TokenData{
		TokenName: tokenData.Name,
		Symbol:    tokenData.Symbol,
		Logo:      tokenData.LogoURI,
		PriceUsd:  price,
		API:       "jup.ag",
	}, nil
}
func geckoterminal(mintAddress string) (*TokenData, error) {
	resp, err := http.Get(fmt.Sprintf("https://api.geckoterminal.com/api/v2/networks/solana/tokens/%s", mintAddress))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tokenData struct {
		Data struct {
			Attributes struct {
				Name     string  `json:"name"`
				Symbol   string  `json:"symbol"`
				ImageURL *string `json:"image_url"`
				PriceUsd float64 `json:"price_usd"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenData); err != nil {
		return nil, err
	}

	return &TokenData{
		TokenName: tokenData.Data.Attributes.Name,
		Symbol:    tokenData.Data.Attributes.Symbol,
		Logo:      tokenData.Data.Attributes.ImageURL,
		PriceUsd:  tokenData.Data.Attributes.PriceUsd,
		API:       "geckoterminal",
	}, nil
}

// Otomatik fiyat güncelleme fonksiyonu
func autoPrice(db *sql.DB) {
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

		var tokenData *TokenData
		switch api {
		case "dexscreener":
			tokenData, err = dexscreener(address)
		case "raydium":
			tokenData, err = raydium(address)
		case "jup.ag":
			tokenData, err = jup(address)
		case "geckoterminal":
			tokenData, err = geckoterminal(address)
		default:
			tokenData, err = dexscreener(address)
		}

		if err != nil {
			continue
		}
		if tokenData != nil && tokenData.PriceUsd > 0 {
			err = updatePrice(db, tokenData.PriceUsd, address)
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
	autoPrice(db)
}

func nullControl(db *sql.DB) {
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

		var tokenData *TokenData
		tokenData, err = dexscreener(address) //dexscreener
		if err == nil && tokenData != nil && tokenData.PriceUsd > 0 {
			log.Printf("Güncellenen Token (dexscreener): %s, Yeni Fiyat: $%.2f", tokenData.TokenName, tokenData.PriceUsd)

			err = updateNullToken(db, tokenData.PriceUsd, "dexscreener", address)
			if err != nil {
				log.Printf("Fiyat güncelleme hatası: %v", err)
			}
			continue
		}
		tokenData, err = jup(address) // Jup
		if err == nil && tokenData != nil && tokenData.PriceUsd > 0 {
			log.Printf("Güncellenen Token (jup.ag): %s, Yeni Fiyat: $%.2f", tokenData.TokenName, tokenData.PriceUsd)

			err = updateNullToken(db, tokenData.PriceUsd, "jup.ag", address)
			if err != nil {
				log.Printf("Fiyat güncelleme hatası: %v", err)
			}
			continue
		}
		tokenData, err = raydium(address) // raydium
		if err == nil && tokenData != nil && tokenData.PriceUsd > 0 {
			log.Printf("Güncellenen Token (raydium): %s, Yeni Fiyat: $%.2f", tokenData.TokenName, tokenData.PriceUsd)

			err = updateNullToken(db, tokenData.PriceUsd, "raydium", address)
			if err != nil {
				log.Printf("Fiyat güncelleme hatası: %v", err)
			}
			continue
		}

		tokenData, err = geckoterminal(address) //geckoterminal
		if err == nil && tokenData != nil && tokenData.PriceUsd > 0 {
			log.Printf("Güncellenen Token (geckoterminal): %s, Yeni Fiyat: $%.2f", tokenData.TokenName, tokenData.PriceUsd)

			err = updateNullToken(db, tokenData.PriceUsd, "geckoterminal", address)
			if err != nil {
				log.Printf("Fiyat güncelleme hatası: %v", err)
			}
			continue
		}

		log.Printf("Fiyat bulunamadı: %s", address)
	}
	log.Printf("Bitti.")
}

func main() {
	// PostgreSQL bağlantısını ayarlıyoruz
	connStr := "host=89.252.131.214 user=postgres password=Washere.123 dbname=wllt port=5432 sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Veritabanı bağlantı hatası: %v", err)
	}
	defer db.Close()

	// Bağlantıyı doğrula
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Bağlantı doğrulama hatası: %v", err)
	}

	fmt.Println("Bağlantı başarılı!")
	//autoPrice(db)
	nullControl(db)
}
