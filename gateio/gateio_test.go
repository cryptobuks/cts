package gateio

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestInit(t *testing.T) {
	Convey("should init successfully", t, func() {
		So(key, ShouldEqual, "")
		So(secret, ShouldEqual, "")
		Init("apikey", "secretkey")
		So(key, ShouldEqual, "apikey")
		So(secret, ShouldEqual, "secretkey")
	})
}

func TestGetPairs(t *testing.T) {
	Convey("should return pair list", t, func() {
		r, err := GetPairs()
		So(err, ShouldBeNil)
		So(r, ShouldNotBeEmpty)
	})
}

func TestTicker(t *testing.T) {
	Convey("should return pair ticker", t, func() {
		Init("key", "secret")
		r, err := Ticker("btc_usdt")
		So(err, ShouldBeNil)
		So(r.Result, ShouldEqual, "true")

		_, err = Ticker("btc_shit")
		So(err, ShouldNotBeNil)
	})
}

func TestMyBalance(t *testing.T) {
	Convey("should return balances unsuccessfully", t, func() {
		Init("key", "secret")
		_, err := MyBalance()
		So(err, ShouldNotBeNil)
	})
}

func TestBuy(t *testing.T) {
	Convey("should buy unsuccessfully", t, func() {
		Init("key", "secret")
		_, err := Buy("doge_usdt", 0.002, 5000)
		So(err, ShouldNotBeNil)
	})
}

func TestSell(t *testing.T) {
	Convey("should buy unsuccessfully", t, func() {
		Init("key", "secret")
		_, err := Sell("doge_usdt", 10000, 0.001)
		So(err, ShouldNotBeNil)
	})
}
