package modules

import (
	"database/sql"
	"encoding/json"
	"fmt"

	//"time"
	"log"
	"strconv"
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

		fmt.Println(wallet.Wallet, "başlanıyor....")
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
			portfolioTotalUsd += nowPrice * parseFloat(token.TokenAmount)
		}
		formattedTotal := formatNumber(portfolioTotalUsd)
		fmt.Println("total token usd:", formattedTotal)
		// Tarih bilgisi alalım (örneğin, güncel tarih)
		//currentDate := time.Now().Format("2006-01-02 15:04:05")
		//
		//// Güncelleme sorgusu
		//updateHistoryQuery := `
		//	UPDATE wallets
		//	SET historyTokenBalance = COALESCE(
		//		historyTokenBalance || jsonb_build_array(jsonb_build_object('price', to_json($1::numeric), 'date', to_json($2::text))),
		//		jsonb_build_array(jsonb_build_object('price', to_json($1::numeric), 'date', to_json($2::text)))
		//	)
		//	WHERE wallet = $3`
		//
		//_, err = client.Exec(updateHistoryQuery, portfolioTotalUsd, currentDate, wallet.Wallet)
		//if err != nil {
		//	log.Printf("Error updating wallet history for %s: %v", wallet.Wallet, err)
		//}
	}

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
func formatNumber(num float64) string {
	// Sayıyı tam sayıya dönüştür
	intValue := int64(num)
	decimalPart := num - float64(intValue)

	// Tam sayı kısmını formatla
	strDecimal := fmt.Sprintf("%.2f", decimalPart)[2:] // Ondalık kısmını al

	// Tam sayının formatlanması (milyon.bin.yüz)
	million := intValue / 1000000
	remaining := intValue % 1000000
	var formatted string
	if million > 0 {
		formatted += strconv.FormatInt(million, 10) + "."
	}

	// Kalan kısmı formatla
	if remaining > 0 {
		// Başındaki sıfırları kaldırmak için 3 basamaklı format
		formatted += fmt.Sprintf("%d.%03d", remaining/1000, remaining%1000)
	} else if million == 0 { // Eğer milyon yoksa kalan kısmı da göstermek için
		formatted += fmt.Sprintf("%03d.000", remaining)
	}

	// Kuruş kısmını ekle
	formatted += "," + strDecimal

	return formatted
}
