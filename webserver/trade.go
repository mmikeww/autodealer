package webserver

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"net/http"
	"strconv"
	"strings"
)

// TradeHandler handleHome is the handler for the '/trade' page request.
func TradeHandler(w http.ResponseWriter, r *http.Request) {
	if err := tpl.ExecuteTemplate(w, "trade.html", nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		logrus.Errorf("error template: %s\n", err)
		return
	}
}

func getTradeResponse(w http.ResponseWriter, r *http.Request) {
	//ctx := r.Context()
	//exchangeResponse, ok := ctx.Value("response").(*transfer.ExchangeWithdrawResponse) // TODO fix
	//if !ok {
	//	logrus.Errorf("Got unexpected response %T instead of *ExchangeWithdrawResponse", exchangeResponse)
	//	//http.Error(w, http.StatusText(http.StatusUnprocessableEntity), http.StatusUnprocessableEntity)
	//	//render.Render(w, r, transfer.ErrWithdawRender(errors.Newf("Failed to renderWithdrawResponse")))
	//	return
	//}
	//
	//render.Render(w, r, exchangeResponse)

	return
}

func TradeCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		exchangeNameReq := chi.URLParam(request, "exchange")
		base := currency.NewCode(chi.URLParam(request, "base"))
		quote := currency.NewCode(chi.URLParam(request, "quote"))
		qtyUSD, err := strconv.ParseFloat(chi.URLParam(request, "qty"), 32)
		if err != nil {
			logrus.Errorf("failed to parse qty %s\n", err)
		}

		p := currency.NewPair(base, quote)
		var assetItem asset.Item

		switch strings.ToLower(chi.URLParam(request, "assetType")); assetItem {
		case "spot":
			assetItem = asset.Spot
		case "futures":
			assetItem = asset.Futures
		default:
			assetItem = asset.Spot
		}

		//var orderType order.Type
		//var side order.Side
		//var postOnly bool
		//switch strings.ToLower(chi.URLParam(request, "orderType")) {
		//case "marketBuy":
		//	orderType = order.Market
		//	side = order.Buy
		//case "limitBuy":
		//	orderType = order.Limit
		//	side = order.Sell
		//	postOnly = true
		//}

		d := GetDealerInstance()
		e, err := d.ExchangeManager.GetExchangeByName(exchangeNameReq)
		if err != nil {
			logrus.Errorf("failed to get exchange: %s\n", exchangeNameReq)
			return
		}

		// try to find out how to enable all pairs??
		d.Settings.EnableAllPairs = true
		d.Settings.EnableCurrencyStateManager = true

		x, err := e.GetEnabledPairs(assetItem)
		if err != nil {
			logrus.Errorf("Get enabled pairs failed %s\n", err)
		}

		logrus.Printf("all parts enabled: %v\n", x)

		if err = e.UpdateCurrencyStates(context.Background(), asset.Spot); err != nil {
			logrus.Errorf("currency state update failed: %s\n", err)
		}

		f, err := e.GetCurrencyStateSnapshot()
		if err != nil {
			logrus.Errorf("currency snapshot update failed: %s\n", err)
		}

		logrus.Info(f)

		if err = e.CanTrade(base, assetItem); err != nil {
			logrus.Errorf("Can not trade: %s\n", err)
		} // currency state fails

		if err = e.CanTradePair(p, assetItem); err != nil {
			logrus.Errorf("can not trade pair %s\n", err)
		}

		price, err := e.UpdateTicker(context.Background(), p, assetItem)

		qty := qtyUSD / price.Ask

		logrus.Printf("quantity %f\n", qty)

		//d.GetActiveOrders(context.Background(), e.GetName(), )

		//e.SetPairs()
		//currency.NewPair()
		//
		//
		//order.Submit{
		//	ImmediateOrCancel: false,
		//	HiddenOrder:       false,
		//	FillOrKill:        false,
		//	PostOnly:          postOnly,
		//	ReduceOnly:        false,
		//	Leverage:          0,
		//	Price:             0,
		//	Amount:            0,
		//	StopPrice:         0,
		//	LimitPriceUpper:   0,
		//	LimitPriceLower:   0,
		//	TriggerPrice:      0,
		//	TargetAmount:      0,
		//	ExecutedAmount:    0,
		//	RemainingAmount:   0,
		//	Fee:               0,
		//	Exchange:          "",
		//	InternalOrderID:   "",
		//	ID:                "",
		//	AccountID:         "",
		//	ClientID:          "",
		//	ClientOrderID:     "",
		//	WalletAddress:     "",
		//	Offset:            "",
		//	Type:              "",
		//	Side:              "",
		//	Status:            "",
		//	AssetType:         "",
		//	Date:              time.Time{},
		//	LastUpdated:       time.Time{},
		//	Pair:              currency.Pair{},
		//	Trades:            nil,
		//}
		ctx := context.WithValue(request.Context(), "response", order.Submit{})
		next.ServeHTTP(w, request.WithContext(ctx))

	})
}
