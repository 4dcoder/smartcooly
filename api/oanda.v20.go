package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/bitly/go-simplejson"
	"github.com/marstau/conver"
	"github.com/marstau/smartcooly/constant"
	"github.com/marstau/smartcooly/model"
)

// OandaV20 the exchange struct of oanda.com v20
type OandaV20 struct {
	stockTypeMap        map[string][2]string
	tradeTypeMap        map[string]string
	tradeTypeAntiMap    map[int]string
	tradeTypeLogMap     map[string]string
	contractTypeAntiMap map[string]string
	leverageMap         map[string]string
	recordsPeriodMap    map[string]string
	minAmountMap        map[string]float64
	records             map[string][]Record
	host                string
	logger              model.Logger
	option              Option

	limit     float64
	lastSleep int64
	lastTimes int64
}

// NewOandaV20 create an exchange struct of okcoin.cn
func NewOandaV20(opt Option) Exchange {
	return &OandaV20{
		stockTypeMap: map[string][2]string{
			"BTC.WEEK/USD":   {"btc_usd", "this_week"},
			"BTC.WEEK2/USD":  {"btc_usd", "next_week"},
			"BTC.MONTH3/USD": {"btc_usd", "quarter"},
			"LTC.WEEK/USD":   {"ltc_usd", "this_week"},
			"LTC.WEEK2/USD":  {"ltc_usd", "next_week"},
			"LTC.MONTH3/USD": {"ltc_usd", "quarter"},
		},
		tradeTypeMap: map[string]string{
			constant.TradeTypeLong:       "1",
			constant.TradeTypeShort:      "2",
			constant.TradeTypeLongClose:  "3",
			constant.TradeTypeShortClose: "4",
		},
		tradeTypeAntiMap: map[int]string{
			1: constant.TradeTypeLong,
			2: constant.TradeTypeShort,
			3: constant.TradeTypeLongClose,
			4: constant.TradeTypeShortClose,
		},
		tradeTypeLogMap: map[string]string{
			constant.TradeTypeLong:       constant.LONG,
			constant.TradeTypeShort:      constant.SHORT,
			constant.TradeTypeLongClose:  constant.LONGCLOSE,
			constant.TradeTypeShortClose: constant.SHORTCLOSE,
		},
		contractTypeAntiMap: map[string]string{
			"this_week": "WEEK",
			"next_week": "WEEK2",
			"quarter":   "MONTH3",
		},
		leverageMap: map[string]string{
			"10": "10",
			"20": "20",
		},
		recordsPeriodMap: map[string]string{
			"M":   "1min",
			"M3":  "3min",
			"M5":  "5min",
			"M15": "15min",
			"M30": "30min",
			"H":   "1hour",
			"H2":  "2hour",
			"H4":  "4hour",
			"H6":  "6hour",
			"H12": "12hour",
			"D":   "1day",
			"D3":  "3day",
			"W":   "1week",
		},
		minAmountMap: map[string]float64{
			"BTC/CNY": 0.01,
			"LTC/CNY": 0.1,
		},
		records: make(map[string][]Record),
		host:    "https://api-fxtrade.oanda.com",
		logger:  model.Logger{TraderID: opt.TraderID, ExchangeType: opt.Type},
		option:  opt,

		limit:     10.0,
		lastSleep: time.Now().UnixNano(),
	}
}

// Log print something to console
func (e *OandaV20) Log(msgs ...interface{}) {
	e.logger.Log(constant.INFO, "", 0.0, 0.0, msgs...)
}

// GetType get the type of this exchange
func (e *OandaV20) GetType() string {
	return e.option.Type
}

// GetName get the name of this exchange
func (e *OandaV20) GetName() string {
	return e.option.Name
}

// SetLimit set the limit calls amount per second of this exchange
func (e *OandaV20) SetLimit(times interface{}) float64 {
	e.limit = conver.Float64Must(times)
	return e.limit
}

// AutoSleep auto sleep to achieve the limit calls amount per second of this exchange
func (e *OandaV20) AutoSleep() {
	now := time.Now().UnixNano()
	interval := 1e+9/e.limit*conver.Float64Must(e.lastTimes) - conver.Float64Must(now-e.lastSleep)
	if interval > 0.0 {
		time.Sleep(time.Duration(conver.Int64Must(interval)))
	}
	e.lastTimes = 0
	e.lastSleep = now
}

// GetMinAmount get the min trade amonut of this exchange
func (e *OandaV20) GetMinAmount(stock string) float64 {
	return e.minAmountMap[stock]
}

func (e *OandaV20) getAuthJSON(method, url string, body interface{}) (statusCode int, resp *simplejson.Json, err error) {
	e.lastTimes++
	bs := []byte{}
	if body != nil {
		if bs, err = json.Marshal(body); err != nil {
			err = fmt.Errorf("[%s %s] HTTP Error Info: %v", method, url, err)
			return
		}
	}
	req, err := http.NewRequest(method, e.host+url, bytes.NewReader(bs))
	if err != nil {
		err = fmt.Errorf("[%s %s] HTTP Error Info: %v", method, url, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.option.SecretKey)
	response, err := client.Do(req)
	rets := []byte{}
	if response == nil {
		err = fmt.Errorf("[%s %s] HTTP Error", method, url)
	} else {
		statusCode = response.StatusCode
		rets, _ = ioutil.ReadAll(response.Body)
		response.Body.Close()
	}
	resp, err = simplejson.NewJson(rets)
	return
}

// GetAccount get the account detail of this exchange
func (e *OandaV20) GetAccount() interface{} {
	statusCode, json, err := e.getAuthJSON("GET", "/v3/accounts/"+e.option.AccessKey+"/summary", nil)
	if err != nil {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetAccount() error, ", err)
		return false
	}
	if statusCode > 200 {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetAccount() error, ", json.Get("errorMessage").Interface())
		return false
	}
	currency := json.GetPath("account", "currency").MustString()
	if currency == "" {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetAccount() error, can not get the currency")
		return false
	}
	return map[string]float64{
		currency:            conver.Float64Must(json.GetPath("account", "marginAvailable").Interface()),
		"Frozen" + currency: 0.0,
	}
}

// GetPositions get the positions detail of this exchange
func (e *OandaV20) GetPositions(stockType string) interface{} {
	stockTypeRaw := strings.ToUpper(stockType)
	stockType = strings.Replace(stockTypeRaw, "/", "_", -1)
	if !strings.Contains(stockType, "_") {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetPositions() error, unrecognized stockType: ", stockType)
		return false
	}
	positions := []Position{}
	statusCode, json, err := e.getAuthJSON("GET", "/v3/accounts/"+e.option.AccessKey+"/positions/"+stockType, nil)
	if err != nil {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetPositions() error, ", err)
		return false
	}
	if statusCode > 200 {
		if json.Get("errorCode").MustString() == "NO_SUCH_POSITION" {
			return positions
		}
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetAccount() error, ", json.Get("errorMessage").Interface())
		return false
	}
	positionJSON := json.GetPath("position", "long")
	amount := conver.Float64Must(positionJSON.Get("units").Interface())
	if amount > 0.0 {
		positions = append(positions, Position{
			Price:         conver.Float64Must(positionJSON.Get("averagePrice").Interface()),
			Leverage:      1,
			Amount:        amount,
			ConfirmAmount: amount,
			FrozenAmount:  0.0,
			Profit:        conver.Float64Must(positionJSON.Get("resettablePL").Interface()),
			ContractType:  "",
			TradeType:     constant.TradeTypeLong,
			StockType:     stockTypeRaw,
		})
	}
	positionJSON = json.GetPath("position", "short")
	amount = conver.Float64Must(positionJSON.Get("units").Interface())
	if amount > 0.0 {
		positions = append(positions, Position{
			Price:         conver.Float64Must(positionJSON.Get("averagePrice").Interface()),
			Leverage:      1,
			Amount:        amount,
			ConfirmAmount: amount,
			FrozenAmount:  0.0,
			Profit:        conver.Float64Must(positionJSON.Get("resettablePL").Interface()),
			ContractType:  "",
			TradeType:     constant.TradeTypeLong,
			StockType:     stockTypeRaw,
		})
	}
	return positions
}

// Trade place an order
func (e *OandaV20) Trade(tradeType string, stockType string, _price, _amount interface{}, msgs ...interface{}) interface{} {
	tradeType = strings.ToUpper(tradeType)
	stockType = strings.ToUpper(stockType)
	price := conver.Float64Must(_price)
	amount := conver.Float64Must(_amount)
	if _, ok := e.tradeTypeMap[tradeType]; !ok {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "Trade() error, unrecognized tradeType: ", tradeType)
		return false
	}
	if _, ok := e.stockTypeMap[stockType]; !ok {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "Trade() error, unrecognized stockType: ", stockType)
		return false
	}
	if len(msgs) < 1 {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "Trade() error, unrecognized leverage")
		return false
	}
	leverage := fmt.Sprint(msgs[0])
	if _, ok := e.leverageMap[leverage]; !ok {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "Trade() error, unrecognized leverage: ", leverage)
		return false
	}
	matchPrice := "match_price=1"
	if price > 0.0 {
		matchPrice = "match_price=0"
	} else {
		price = 0.0
	}
	params := []string{
		"symbol=" + e.stockTypeMap[stockType][0],
		"contract_type=" + e.stockTypeMap[stockType][1],
		fmt.Sprintf("price=%f", price),
		fmt.Sprintf("amount=%f", amount),
		"type=" + e.tradeTypeMap[tradeType],
		matchPrice,
		"lever_rate=" + leverage,
	}
	_, json, err := e.getAuthJSON("GET", e.host+"future_trade.do", params)
	if err != nil {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "Trade() error, ", err)
		return false
	}
	if result := json.Get("result").MustBool(); !result {
		err = fmt.Errorf("Trade() error, the error number is %v", json.Get("error_code").MustInt())
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "Trade() error, ", err)
		return false
	}
	e.logger.Log(e.tradeTypeLogMap[tradeType], stockType, price, amount, msgs[2:]...)
	return fmt.Sprint(json.Get("order_id").Interface())
}

// GetOrder get details of an order
func (e *OandaV20) GetOrder(stockType, id string) interface{} {
	stockType = strings.ToUpper(stockType)
	if _, ok := e.stockTypeMap[stockType]; !ok {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetOrder() error, unrecognized stockType: ", stockType)
		return false
	}
	params := []string{
		"symbol=" + e.stockTypeMap[stockType][0],
		"contract_type=" + e.stockTypeMap[stockType][1],
		"order_id=" + id,
	}
	_, json, err := e.getAuthJSON("GET", e.host+"future_orders_info.do", params)
	if err != nil {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetOrder() error, ", err)
		return false
	}
	if result := json.Get("result").MustBool(); !result {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetOrder() error, the error number is ", json.Get("error_code").MustInt())
		return false
	}
	ordersJSON := json.Get("orders")
	if len(ordersJSON.MustArray()) > 0 {
		orderJSON := ordersJSON.GetIndex(0)
		return Order{
			ID:         fmt.Sprint(orderJSON.Get("order_id").Interface()),
			Price:      orderJSON.Get("price").MustFloat64(),
			Amount:     orderJSON.Get("amount").MustFloat64(),
			DealAmount: orderJSON.Get("deal_amount").MustFloat64(),
			Fee:        orderJSON.Get("fee").MustFloat64(),
			TradeType:  e.tradeTypeAntiMap[orderJSON.Get("type").MustInt()],
			StockType:  stockType,
		}
	}
	return false
}

// GetOrders get all unfilled orders
func (e *OandaV20) GetOrders(stockType string) interface{} {
	stockType = strings.ToUpper(stockType)
	orders := []Order{}
	if _, ok := e.stockTypeMap[stockType]; !ok {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetOrders() error, unrecognized stockType: ", stockType)
		return false
	}
	params := []string{
		"symbol=" + e.stockTypeMap[stockType][0],
		"contract_type=" + e.stockTypeMap[stockType][1],
		"status=1",
		"order_id=-1",
		"current_page=1",
		"page_length=50",
	}
	_, json, err := e.getAuthJSON("GET", e.host+"future_order_info.do", params)
	if err != nil {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetOrders() error, ", err)
		return false
	}
	if result := json.Get("result").MustBool(); !result {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetOrders() error, the error number is ", json.Get("error_code").MustInt())
		return false
	}
	ordersJSON := json.Get("orders")
	count := len(ordersJSON.MustArray())
	for i := 0; i < count; i++ {
		orderJSON := ordersJSON.GetIndex(i)
		orders = append(orders, Order{
			ID:         fmt.Sprint(orderJSON.Get("order_id").Interface()),
			Price:      orderJSON.Get("price").MustFloat64(),
			Amount:     orderJSON.Get("amount").MustFloat64(),
			DealAmount: orderJSON.Get("deal_amount").MustFloat64(),
			Fee:        orderJSON.Get("fee").MustFloat64(),
			TradeType:  e.tradeTypeAntiMap[orderJSON.Get("type").MustInt()],
			StockType:  stockType,
		})
	}
	return orders
}

// GetTrades get all filled orders recently
func (e *OandaV20) GetTrades(stockType string) interface{} {
	stockType = strings.ToUpper(stockType)
	orders := []Order{}
	if _, ok := e.stockTypeMap[stockType]; !ok {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetTrades() error, unrecognized stockType: ", stockType)
		return false
	}
	params := []string{
		"symbol=" + e.stockTypeMap[stockType][0],
		"contract_type=" + e.stockTypeMap[stockType][1],
		"status=2",
		"order_id=-1",
		"current_page=1",
		"page_length=50",
	}
	_, json, err := e.getAuthJSON("GET", e.host+"future_order_info.do", params)
	if err != nil {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetTrades() error, ", err)
		return false
	}
	if result := json.Get("result").MustBool(); !result {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetTrades() error, the error number is ", json.Get("error_code").MustInt())
		return false
	}
	ordersJSON := json.Get("orders")
	count := len(ordersJSON.MustArray())
	for i := 0; i < count; i++ {
		orderJSON := ordersJSON.GetIndex(i)
		orders = append(orders, Order{
			ID:         fmt.Sprint(orderJSON.Get("order_id").Interface()),
			Price:      orderJSON.Get("price").MustFloat64(),
			Amount:     orderJSON.Get("amount").MustFloat64(),
			DealAmount: orderJSON.Get("deal_amount").MustFloat64(),
			Fee:        orderJSON.Get("fee").MustFloat64(),
			TradeType:  e.tradeTypeAntiMap[orderJSON.Get("type").MustInt()],
			StockType:  stockType,
		})
	}
	return orders
}

// CancelOrder cancel an order
func (e *OandaV20) CancelOrder(order Order) bool {
	params := []string{
		"symbol=" + e.stockTypeMap[order.StockType][0],
		"order_id=" + order.ID,
		"contract_type=" + e.stockTypeMap[order.StockType][1],
	}
	_, json, err := e.getAuthJSON("GET", e.host+"future_cancel.do", params)
	if err != nil {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "CancelOrder() error, ", err)
		return false
	}
	if result := json.Get("result").MustBool(); !result {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "CancelOrder() error, the error number is ", json.Get("error_code").MustInt())
		return false
	}
	e.logger.Log(constant.CANCEL, order.StockType, order.Price, order.Amount-order.DealAmount, order)
	return true
}

// getTicker get market ticker & depth
func (e *OandaV20) getTicker(stockType string, sizes ...interface{}) (ticker Ticker, err error) {
	stockType = strings.ToUpper(stockType)
	if _, ok := e.stockTypeMap[stockType]; !ok {
		err = fmt.Errorf("GetTicker() error, unrecognized stockType: %+v", stockType)
		return
	}
	size := 20
	if len(sizes) > 0 && conver.IntMust(sizes[0]) > 0 {
		size = conver.IntMust(sizes[0])
	}
	resp, err := get(fmt.Sprintf("%vfuture_depth.do?symbol=%v&contract_type=%v&size=%v", e.host, e.stockTypeMap[stockType][0], e.stockTypeMap[stockType][1], size))
	if err != nil {
		err = fmt.Errorf("GetTicker() error, %+v", err)
		return
	}
	json, err := simplejson.NewJson(resp)
	if err != nil {
		err = fmt.Errorf("GetTicker() error, %+v", err)
		return
	}
	depthsJSON := json.Get("bids")
	for i := 0; i < len(depthsJSON.MustArray()); i++ {
		depthJSON := depthsJSON.GetIndex(i)
		ticker.Bids = append(ticker.Bids, OrderBook{
			Price:  depthJSON.GetIndex(0).MustFloat64(),
			Amount: depthJSON.GetIndex(1).MustFloat64(),
		})
	}
	depthsJSON = json.Get("asks")
	for i := len(depthsJSON.MustArray()); i > 0; i-- {
		depthJSON := depthsJSON.GetIndex(i - 1)
		ticker.Asks = append(ticker.Asks, OrderBook{
			Price:  depthJSON.GetIndex(0).MustFloat64(),
			Amount: depthJSON.GetIndex(1).MustFloat64(),
		})
	}
	if len(ticker.Bids) < 1 || len(ticker.Asks) < 1 {
		err = fmt.Errorf("GetTicker() error, can not get enough Bids or Asks")
		return
	}
	ticker.Buy = ticker.Bids[0].Price
	ticker.Sell = ticker.Asks[0].Price
	ticker.Mid = (ticker.Buy + ticker.Sell) / 2
	return
}

// GetTicker get market ticker & depth
func (e *OandaV20) GetTicker(stockType string, sizes ...interface{}) interface{} {
	ticker, err := e.getTicker(stockType, sizes...)
	if err != nil {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, err)
		return false
	}
	return ticker
}

// GetRecords get candlestick data
func (e *OandaV20) GetRecords(stockType, period string, sizes ...interface{}) interface{} {
	stockType = strings.ToUpper(stockType)
	if _, ok := e.stockTypeMap[stockType]; !ok {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetRecords() error, unrecognized stockType: ", stockType)
		return false
	}
	if _, ok := e.recordsPeriodMap[period]; !ok {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetRecords() error, unrecognized period: ", period)
		return false
	}
	size := 200
	if len(sizes) > 0 && conver.IntMust(sizes[0]) > 0 {
		size = conver.IntMust(sizes[0])
	}
	resp, err := get(fmt.Sprintf("%vfuture_kline.do?symbol=%v&contract_type=%v&type=%v&size=%v", e.host, e.stockTypeMap[stockType][0], e.stockTypeMap[stockType][1], e.recordsPeriodMap[period], size))
	if err != nil {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetRecords() error, ", err)
		return false
	}
	json, err := simplejson.NewJson(resp)
	if err != nil {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetRecords() error, ", err)
		return false
	}
	timeLast := int64(0)
	if len(e.records[period]) > 0 {
		timeLast = e.records[period][len(e.records[period])-1].Time
	}
	recordsNew := []Record{}
	for i := len(json.MustArray()); i > 0; i-- {
		recordJSON := json.GetIndex(i - 1)
		recordTime := recordJSON.GetIndex(0).MustInt64() / 1000
		if recordTime > timeLast {
			recordsNew = append([]Record{{
				Time:   recordTime,
				Open:   recordJSON.GetIndex(1).MustFloat64(),
				High:   recordJSON.GetIndex(2).MustFloat64(),
				Low:    recordJSON.GetIndex(3).MustFloat64(),
				Close:  recordJSON.GetIndex(4).MustFloat64(),
				Volume: recordJSON.GetIndex(5).MustFloat64(),
			}}, recordsNew...)
		} else if timeLast > 0 && recordTime == timeLast {
			e.records[period][len(e.records[period])-1] = Record{
				Time:   recordTime,
				Open:   recordJSON.GetIndex(1).MustFloat64(),
				High:   recordJSON.GetIndex(2).MustFloat64(),
				Low:    recordJSON.GetIndex(3).MustFloat64(),
				Close:  recordJSON.GetIndex(4).MustFloat64(),
				Volume: recordJSON.GetIndex(5).MustFloat64(),
			}
		} else {
			break
		}
	}
	e.records[period] = append(e.records[period], recordsNew...)
	if len(e.records[period]) > size {
		e.records[period] = e.records[period][len(e.records[period])-size : len(e.records[period])]
	}
	return e.records[period]
}


func (e *OandaV20) ExchangeRate(count string,stockType string) string {
	return ""
}
