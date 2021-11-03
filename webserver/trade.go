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
	"time"
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
		p, err := currency.NewPairFromString(chi.URLParam(request, "pair"))
		if err != nil {
			logrus.Errorf("failed to parse pair: %s\n", chi.URLParam(request, "pair"))
		}

		qtyUSD, err := strconv.ParseFloat(chi.URLParam(request, "qty"), 32)
		if err != nil {
			logrus.Errorf("failed to parse qty %s\n", err)
		}

		assetItem, err := asset.New(chi.URLParam(request, "assetType"))
		if err != nil {
			logrus.Errorf("failed to parse assetType %s\n", err)
		}

		var orderType order.Type
		var side order.Side
		var postOnly bool

		switch chi.URLParam(request, "orderType") {
		case "marketBuy":
			orderType = order.Market
			side = order.Bid
		case "limitBuy":
			orderType = order.Limit
			side = order.Ask
			postOnly = true
		}

		logrus.Printf("order side %s\n", side)

		d := GetDealerInstance()
		e, err := d.ExchangeManager.GetExchangeByName(exchangeNameReq)
		if err != nil {
			logrus.Errorf("failed to get exchange: %s\n", exchangeNameReq)
			return
		}

		// try to find out how to enable all pairs??
		d.Settings.EnableAllPairs = true
		d.Settings.EnableCurrencyStateManager = true

		price, err := e.UpdateTicker(context.Background(), p, assetItem)
		if err != nil {
			logrus.Errorf("failed to update ticker %s\n", err)
		}

		qty := qtyUSD / price.Ask
		logrus.Printf("%s quantity %f\n", p.String(), qty)

		o := order.Submit{
			ImmediateOrCancel: false,
			HiddenOrder:       false,
			FillOrKill:        false,
			PostOnly:          postOnly,
			ReduceOnly:        false,
			Leverage:          0,
			Price:             price.Ask,
			Amount:            qty,
			StopPrice:         0,
			LimitPriceUpper:   0,
			LimitPriceLower:   0,
			TriggerPrice:      0,
			TargetAmount:      0,
			ExecutedAmount:    0,
			RemainingAmount:   0,
			Fee:               0,
			Exchange:          e.GetName(),
			InternalOrderID:   "",
			ID:                "",
			AccountID:         "",
			ClientID:          "",
			ClientOrderID:     "",
			WalletAddress:     "",
			Offset:            "",
			Type:              orderType,
			Side:              side,
			Status:            "",
			AssetType:         assetItem,
			Date:              time.Now(),
			LastUpdated:       time.Time{},
			Pair:              p,
			Trades:            nil,
		}

		if err = o.Validate(); err != nil {
			logrus.Errorf("failed to validate order: %s\n", err)
		}

		orderResponse, err := e.SubmitOrder(context.Background(), &o)
		if err != nil {
			logrus.Errorf("submit order failed: %s\n", err)
		}
		logrus.Printf("order response ID %s placed %t", orderResponse.OrderID, orderResponse.IsOrderPlaced)

		ctx := context.WithValue(request.Context(), "response", order.Submit{})
		next.ServeHTTP(w, request.WithContext(ctx))
	})
}
