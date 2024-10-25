package modules

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strconv"
	"time"
)

type Token struct {
	Address     string `json:"address"`
	TokenAmount string `json:"tokenAmount"`
}

type Wallet struct {
	Wallet              string          `json:"wallet"`
	Tokens              Tokens          `json:"tokens"`
	SolAmount           float64         `json:"solamount"`           // Numeric alan
	WalletName          string          `json:"walletName"`          // Text alan
	HistorySolBalance   json.RawMessage `json:"historySolBalance"`   // JSONB alan
	HistoryTokenBalance json.RawMessage `json:"historyTokenBalance"` // JSONB alanı
}
type Tokens struct {
	Portfolio []Token `json:"portfolio"`
}

func UpdateWallets(db *sql.DB) error {
	startTime := time.Now()
	client := db // db bağlantısını kullanıyoruz

	walletQuery := `SELECT wallet, tokens FROM wallets`
	rows, err := client.Query(walletQuery)
	if err != nil {
		log.Fatalf("Error querying wallets: %v", err)
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var wallet Wallet
		var tokensJson []byte // tokens alanını []byte olarak okuyacağız

		// Tokenları []byte olarak al
		if err := rows.Scan(&wallet.Wallet, &tokensJson); err != nil {
			log.Fatalf("Error scanning wallet: %v", err)
			return err
		}

		// JSONB alanını Go yapısına dönüştürelim
		if err := json.Unmarshal(tokensJson, &wallet.Tokens); err != nil {
			log.Fatalf("Error unmarshalling tokens: %v", err)
			return err
		}
		portfolioTotalUsd := 0.0
		for _, token := range wallet.Tokens.Portfolio {
			if token.TokenAmount == "0" {
				continue
			}
			updateQuery := `SELECT nowprice FROM tokens WHERE address = $1`
			var nowPrice float64
			err := client.QueryRow(updateQuery, token.Address).Scan(&nowPrice)
			if err != nil {
				continue
			}

			// NaN kontrolü
			if math.IsNaN(nowPrice) {
				portfolioTotalUsd += 0.0
			} else {
				portfolioTotalUsd += nowPrice * parseFloat(token.TokenAmount)
			}
		}

		if portfolioTotalUsd == 0.0 {
			fmt.Printf("Wallet %s has no token balance to update.\n", wallet.Wallet)
			continue
		}
		// Tarih bilgisi
		currentDate := time.Now().Format("2006-01-02T15:04:05Z07:00") // ISO 8601 formatında tarih
		price := portfolioTotalUsd                                    // Burada istediğiniz fiyatı kullanabilirsiniz

		// Güncelleme sorgusu
		updateHistoryQuery := `
	UPDATE wallets
	SET  historytokenbalance = COALESCE(
		        historytokenbalance || jsonb_build_array(jsonb_build_object('price', to_json($1::numeric), 'date', to_json($2::text))),
		        jsonb_build_array(jsonb_build_object('price', to_json($1::numeric), 'date', to_json($2::text)))
		    )
	WHERE wallet = $3`

		_, err = client.Exec(updateHistoryQuery, price, currentDate, wallet.Wallet)
		if err != nil {
			log.Printf("Error updating wallet history for %s: %v", wallet.Wallet, err)
		}
	}
	duration := time.Since(startTime)
	fmt.Printf("İşlem süresi: %d dakika %d saniye.\n", int(duration.Minutes()), int(duration.Seconds())%60)
	return nil
}

func parseFloat(s string) float64 {
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		log.Printf("Error parsing float: %v", err)
		return 0
	}
	return val
}

// formatNumber fonksiyonu, sayıyı istediğiniz formatta döndürür
