package hooks

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// TokenData yapısını tanımlayın
type TokenData struct {
	TokenName string  `json:"tokenName"`
	Symbol    string  `json:"symbol"`
	Logo      *string `json:"logo"`
	PriceUsd  float64 `json:"priceUsd"`
	API       string  `json:"api"`
}

// dexscreener fonksiyonu
func Dexscreener(mintAddress string) (*TokenData, error) {
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
				return nil, fmt.Errorf("price not found")
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
func Raydium(mintAddress string) (*TokenData, error) {
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
			if data.Name == "Slurp" {
				fmt.Printf("token: %s, raydium price: %.9f \n", data.Name, price)
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
func Jup(mintAddress string) (*TokenData, error) {
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
func Geckoterminal(mintAddress string) (*TokenData, error) {
	resp, err := http.Get(fmt.Sprintf("https://api.geckoterminal.com/api/v2/networks/solana/tokens/%s", mintAddress))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// Yanıtın başarılı olup olmadığını kontrol edin
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API isteği başarısız oldu: %s", resp.Status)
	}
	//bodyBytes, err := io.ReadAll(resp.Body)
	//if err != nil {
	//	return nil, err
	//}
	//
	//// Yanıt gövdesini konsola yazdırın
	//fmt.Println("API yanıtı:", string(bodyBytes))
	var tokenData struct {
		Data struct {
			Attributes struct {
				Name     string  `json:"name"`
				Symbol   string  `json:"symbol"`
				ImageURL *string `json:"image_url"`
				PriceUsd string  `json:"price_usd"`
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
		PriceUsd:  parsePrice(tokenData.Data.Attributes.PriceUsd),
		API:       "geckoterminal",
	}, nil
}

func parsePrice(price string) float64 {
	priceFloat, _ := strconv.ParseFloat(price, 64)
	return priceFloat
}
