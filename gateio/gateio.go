package gateio

import (
	"crypto/hmac"
	"crypto/sha512"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	jsoniter "github.com/json-iterator/go"
)

type (
	Balance struct {
		Result    string
		Available map[string]string // 可用
		Locked    map[string]string // 已锁定
	}

	Asset struct {
		Balance  float64
		Total    float64
		TotalCNY float64
	}

	Pair struct {
		Result        string
		PercentChange float64 // 涨跌百分比
		Last          float64 // 最新成交价
		LowestAsk     float64 // 卖方最低价
		HighestBid    float64 // 买方最高价
		BaseVolume    float64 // 交易量
		QuoteVolume   float64 // 兑换货币交易量
		High24hr      float64 // 24 小时最高价
		Low24hr       float64 // 24 小时最低价
	}

	Trade struct {
		Result  string
		Code    int32
		Message string
		OrderID uint64 `json:"orderNumber"` // 订单 ID
	}

	Order struct {
		TradeID   string `json:"tradeID"`
		OrderID   string `json:"orderNumber"`
		Currency  string `json:"pair"`
		Type      string
		Rate      string
		Amount    string
		Total     float64
		Date      string
		Timestamp string `json:"time_unix"`
	}

	gateioError struct {
		// Response:
		// true		success
		// false	fail
		//
		// Result 有可能是字符串 "true"，也有可能是布尔值 true
		// cancelAllOrders 接口会返回布尔值
		Result interface{}

		// Error code:
		// 0	Success
		// 1	Invalid request
		// 2	Invalid version
		// 3	Invalid request
		// 4	Too many attempts
		// 5,6	Invalid sign
		// 7	Currency is not supported
		// 8,9	Currency is not supported
		// 10	Verified failed
		// 11	Obtaining address failed
		// 12	Empty params
		// 13	Internal error, please report to administrator
		// 14	Invalid user
		// 15	Cancel order too fast, please wait 1 min and try again
		// 16	Invalid order id or order is already closed
		// 17	Invalid orderid
		// 18	Invalid amount
		// 19	Not permitted or trade is disabled
		// 20	Your order size is too small
		// 21	You don't have enough fund
		Code    int32
		Message string
	}
)

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary

	key    string // your api key
	secret string // your secret key
)

const (
	CancelTypeSell = 0
	CancelTypeBuy  = 1
	CancelTypeAll  = -1
)

// Init set apikey and secretkey
func Init(apikey, secretkey string) {
	key = apikey
	secret = secretkey
}

// GetPairs return Return all the trading pairs supported by gate.io
func GetPairs() ([]string, error) {
	bs, err := req("GET", "http://data.gate.io/api2/1/pairs", "")
	if err != nil {
		return nil, err
	}

	var l []string
	err = json.Unmarshal(bs, &l)
	if err != nil {
		if err := gateioErrorHandle(bs, nil); err != nil {
			return nil, err
		}
		return nil, err
	}

	return l, nil
}

// Ticker returns the current ticker for the selected currency,
// cached in 10 seconds.
func Ticker(currency string) (*Pair, error) {
	bs, err := req("GET",
		fmt.Sprintf("http://data.gate.io/api2/1/ticker/%s", currency), "")
	if err := gateioErrorHandle(bs, err); err != nil {
		return nil, err
	}

	p := Pair{}
	err = json.Unmarshal(bs, &p)
	if err != nil {
		return nil, err
	}

	return &p, nil
}

// MyBalance return account balances
func MyBalance() (*Balance, error) {
	bs, err := req("POST", "https://api.gate.io/api2/1/private/balances", "")
	if err := gateioErrorHandle(bs, err); err != nil {
		return nil, err
	}

	b := Balance{}
	err = json.Unmarshal(bs, &b)
	if err != nil {
		return nil, err
	}

	return &b, nil
}

// MyAsset return account assets
func MyAsset() (*Asset, error) {
	b, err := MyBalance()
	if err != nil {
		return nil, err
	}

	a := Asset{}
	for k, v := range b.Available {
		n, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, err
		}

		if k == "USDT" {
			a.Balance = n
			a.Total += n
		} else {
			t, err := Ticker(strings.ToLower(k) + "_usdt")
			if err != nil {
				return nil, err
			}

			a.Total += t.Last * n
		}
	}

	r, err := Rate()
	if err != nil {
		return nil, err
	}

	a.TotalCNY = r * a.Total

	return &a, nil
}

// Rate return exchange rate of USD/CNY
func Rate() (float64, error) {
	bs, err := req("GET", "http://data.gate.io/api2/1/ticker/usdt_cny", "")
	if err := gateioErrorHandle(bs, err); err != nil {
		return 0, err
	}

	p := struct{ Last string }{}
	err = json.Unmarshal(bs, &p)
	if err != nil {
		return 0, err
	}

	f, err := strconv.ParseFloat(p.Last, 64)
	if err != nil {
		return 0, err
	}

	return f, nil
}

// Buy place order buy
func Buy(currency string, price float64, amount float64) (*Trade, error) {
	params := fmt.Sprintf("currencyPair=%s&rate=%f&amount=%f",
		currency, price, amount)

	bs, err := req("POST", "https://api.gate.io/api2/1/private/buy", params)
	if err := gateioErrorHandle(bs, err); err != nil {
		return nil, err
	}

	t := Trade{}
	err = json.Unmarshal(bs, &t)
	if err != nil {
		return nil, err
	}

	return &t, nil
}

// Sell place order sell
func Sell(currency string, price float64, amount float64) (*Trade, error) {
	params := fmt.Sprintf("currencyPair=%s&rate=%f&amount=%f",
		currency, price, amount)

	bs, err := req("POST", "https://api.gate.io/api2/1/private/sell", params)
	if err := gateioErrorHandle(bs, err); err != nil {
		return nil, err
	}

	t := Trade{}
	err = json.Unmarshal(bs, &t)
	if err != nil {
		return nil, err
	}

	return &t, nil
}

// Cancel cancel all orders
func Cancel(currency string, cancelType int8) error {
	params := fmt.Sprintf("currencyPair=%s&type=%d", currency, cancelType)
	url := "https://api.gate.io/api2/1/private/cancelAllOrders"

	bs, err := req("POST", url, params)
	err = gateioErrorHandle(bs, err)

	return err
}

// LatestOrder return latest order of my last 24h trades
func LatestOrder(currency string) (*Order, error) {
	url := "https://api.gate.io/api2/1/private/tradeHistory"
	bs, err := req("POST", url, "currencyPair="+currency)
	err = gateioErrorHandle(bs, err)
	if err := gateioErrorHandle(bs, err); err != nil {
		return nil, err
	}

	r := struct{ Trades []*Order }{}

	err = json.Unmarshal(bs, &r)
	if err != nil {
		return nil, err
	}

	if len(r.Trades) != 0 {
		return r.Trades[0], nil
	}

	return &Order{}, nil
}

func sign(params string) (string, error) {
	key := []byte(secret)

	mac := hmac.New(sha512.New, key)
	mac.Write([]byte(params))

	return fmt.Sprintf("%x", mac.Sum(nil)), nil
}

func req(method string, url string, param string) ([]byte, error) {
	client := &http.Client{}

	req, err := http.NewRequest(method, url, strings.NewReader(param))
	if err != nil {
		return nil, err
	}

	s, err := sign(param)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("key", key)
	req.Header.Set("sign", s)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return bs, nil
}

func gateioErrorHandle(bs []byte, err error) error {
	if err != nil {
		return err
	}

	e := gateioError{}

	err = json.Unmarshal(bs, &e)
	if err != nil {
		return err
	}

	if e.Code != 0 {
		return errors.New(fmt.Sprintf("Code: %d, %s", e.Code, e.Message))
	}

	return nil
}
