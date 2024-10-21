package GetPortfolio

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"microservices/hooks"
	"time"

	"github.com/blocto/solana-go-sdk/common"
	"github.com/blocto/solana-go-sdk/rpc"
	_ "github.com/lib/pq"
)

type TokenAccount struct {
	Mint        string `json:"mint"`
	TokenAmount struct {
		UiAmountString string `json:"uiAmountString"`
	} `json:"tokenAmount"`
}

func getPortfolio(walletAddress string, db *sql.DB) error {
	// RPC client'ı fonksiyon içinde tanımlayın
	client := rpc.New(rpc.WithEndpoint("https://palpable-long-season.solana-mainnet.quiknode.pro/446ebd93d3a558608cbad70a18a12ab59439bd9c/"))
	startTime := time.Now()

	// walletAddress string'ini PublicKey formatına dönüştürme
	ownerPubKey := common.PublicKeyFromString(walletAddress)

	// Token hesaplarını alma
	ctx := context.Background() // Yeni bir context oluşturun
	tokenAccountsResp, err := client.GetTokenAccountsByOwner(ctx, ownerPubKey.String(), rpc.GetTokenAccountsByOwnerConfigFilter{
		ProgramId: "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA", // TOKEN_PROGRAM_ID
	})
	if err != nil {
		log.Fatalf("Error fetching token accounts: %v", err)
		return err
	}

	if len(tokenAccountsResp.Result.Value) == 0 {
		log.Println("Bu cüzdanda hiç SPL token yok.")
		return nil
	}

	var portfolio []map[string]interface{}
	for _, acc := range tokenAccountsResp.Result.Value {
		data, ok := acc.Account.Data.([]interface{}) // Data'nın bir slice olduğunu varsayıyoruz
		if !ok || len(data) == 0 {
			log.Println("Geçersiz token hesap verisi.")
			continue
		}

		// Base64 çözümlemesi ve Unmarshal
		dataBytes, err := base64.StdEncoding.DecodeString(data[0].(string)) // Base64 çözümlemesi
		if err != nil {
			log.Printf("Base64 çözümleme hatası: %v", err)
			continue
		}

		var tokenAcc TokenAccount
		err = json.Unmarshal(dataBytes, &tokenAcc)
		if err != nil {
			log.Printf("Hata: Token verileri parse edilemedi: %v", err)
			continue
		}

		if tokenAcc.TokenAmount.UiAmountString != "0" {
			var data *hooks.TokenData
			mint := tokenAcc.Mint

			// Veritabanı sorgusu
			query := `SELECT tokenname, symbol, logo FROM public.tokens WHERE address = $1`
			row := db.QueryRow(query, mint)
			var tokenName, symbol string
			var logo *string

			// Veritabanı kaydı varsa portfolyo dizisine ekle
			if row.Scan(&tokenName, &symbol, &logo) == nil {
				portfolio = append(portfolio, map[string]interface{}{
					"address": mint,
					"tokenMeta": map[string]interface{}{
						"name":   tokenName,
						"symbol": symbol,
						"logo":   logo,
					},
					"tokenAmount": tokenAcc.TokenAmount.UiAmountString,
				})
				continue
			}

			// API çağrıları ve portföy ekleme
			data, err = hooks.Dexscreener(mint)
			if err == nil && data != nil {
				insertTokenData(db, mint, data)
				portfolio = append(portfolio, preparePortfolioEntry(mint, data, tokenAcc.TokenAmount.UiAmountString))
				continue
			}

			data, err = hooks.Raydium(mint)
			if err == nil && data != nil {
				insertTokenData(db, mint, data)
				portfolio = append(portfolio, preparePortfolioEntry(mint, data, tokenAcc.TokenAmount.UiAmountString))
				continue
			}

			data, err = hooks.Jup(mint)
			if err == nil && data != nil {
				insertTokenData(db, mint, data)
				portfolio = append(portfolio, preparePortfolioEntry(mint, data, tokenAcc.TokenAmount.UiAmountString))
				continue
			}

			data, err = hooks.Geckoterminal(mint)
			if err == nil && data != nil {
				insertTokenData(db, mint, data)
				portfolio = append(portfolio, preparePortfolioEntry(mint, data, tokenAcc.TokenAmount.UiAmountString))
				continue
			}
		}
	}

	// Portföy verilerini güncelleme
	updateQuery := `UPDATE public.wallets SET tokens = $1 WHERE wallet = $2`
	_, err = db.Exec(updateQuery, portfolio, walletAddress)
	if err != nil {
		log.Printf("Veritabanı güncelleme hatası: %v", err)
	}

	// İşlem süresi hesaplama
	endTime := time.Now()
	timeDiff := endTime.Sub(startTime)
	fmt.Printf("İşlem süresi: %v dakika %v saniye.\n", int(timeDiff.Minutes()), int(timeDiff.Seconds())%60)

	return nil
}

func insertTokenData(db *sql.DB, mint string, data *hooks.TokenData) error {
	query := `
		INSERT INTO tokens (address, tokenname, symbol, logo, nowprice, history, api)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := db.Exec(query, mint, data.TokenName, data.Symbol, data.Logo, data.PriceUsd, "{}", data.API)
	if err != nil {
		log.Printf("Veritabanı ekleme hatası: %v", err)
	}
	return err
}

func preparePortfolioEntry(mint string, data *hooks.TokenData, tokenAmount string) map[string]interface{} {
	return map[string]interface{}{
		"address": mint,
		"tokenMeta": map[string]interface{}{
			"name":   data.TokenName,
			"symbol": data.Symbol,
			"logo":   data.Logo,
			"api":    data.API,
		},
		"tokenAmount": tokenAmount,
	}
}
