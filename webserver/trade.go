package webserver

import (
	"context"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/romanornr/autodealer/orderbuilder"
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

// OrderResponse is the response for the '/order' request.
type OrderResponse struct {
	Response  order.SubmitResponse `json:"response"`
	Order     order.Submit         `json:"order"`
	Pair      string               `json:"pair"`
	QtyUSD    float64              `json:"qtyUSD"`
	Qty       float64              `json:"qty"`
	Price     float64              `json:"price"`
	Timestamp time.Time            `json:"timestamp"`
}

// getTradeResponse returns the trade response
func getTradeResponse(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	response, ok := ctx.Value("response").(*OrderResponse) // TODO fix
	if !ok {
		logrus.Errorf("Got unexpected response %T\n", response)
		http.Error(w, http.StatusText(http.StatusUnprocessableEntity), http.StatusUnprocessableEntity)
		render.JSON(w, r, ErrRender(errors.New("failed to get trade response")))
		return
	}
	render.JSON(w, r, response)
}

// TradeCtx Handler handleHome is the handler for the '/trade' page request.
// trade/{exchange}/{pair}/{qty}/{assetType}/{orderType}/{side}
func TradeCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		exchangeNameReq := chi.URLParam(request, "exchange")
		p, err := currency.NewPairFromString(chi.URLParam(request, "pair"))
		if err != nil {
			logrus.Errorf("failed to parse pair: %s\n", chi.URLParam(request, "pair"))
		}

		assetItem := asset.Item(chi.URLParam(request, "assetType"))
		if !assetItem.IsValid() {
			logrus.Errorf("failed to parse assetType: %s\n", chi.URLParam(request, "assetType"))
		}

		side, err := order.StringToOrderSide(chi.URLParam(request, "side"))
		if err != nil {
			logrus.Errorf("failed to parse side: %s\n", chi.URLParam(request, "side"))
		}

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

		qtyUSD, err := strconv.ParseFloat(chi.URLParam(request, "qty"), 64)
		if err != nil {
			logrus.Errorf("failed to parse qty %s\n", err)
		}

		orderType, err := order.StringToOrderType(chi.URLParam(request, "orderType"))
		if err != nil {
			logrus.Errorf("failed to parse orderType %s\n", err)
		}

		qty := qtyUSD / price.Last
		subAccount, err := GetSubAccountByID(e, "")
		orderBuilder := orderbuilder.NewOrderBuilder()
		orderBuilder.
			AtExchange(e.GetName()).
			ForCurrencyPair(p).
			WithAssetType(assetItem).
			ForPrice(price.Last).
			WithAmount(qty).
			UseOrderType(orderType).
			SetSide(side).
			WithPostOnly(false).
			UseImmediateOrCancel(false).
			ForAccountID(subAccount.ID)

		newOrder, err := orderBuilder.Build()
		if err != nil {
			logrus.Errorf("failed to build order %s\n", err)
		}

		logrus.Printf("%s quantity %f\n", p.String(), qty)

		submitResponse, err := d.SubmitOrderUD(context.Background(), e.GetName(), *newOrder, nil) //e.SubmitOrder(context.Background(), &o)
		if err != nil {
			logrus.Errorf("submit order failed: %s\n", err)
		}
		logrus.Printf("order response ID %s placed %t", submitResponse.OrderID, submitResponse.IsOrderPlaced)

		response := OrderResponse{
			Response:  submitResponse,
			Order:     *newOrder,
			Pair:      p.String(),
			QtyUSD:    qtyUSD,
			Qty:       qty,
			Price:     price.Ask,
			Timestamp: time.Now(),
		}

		ctx := context.WithValue(request.Context(), "response", &response)
		next.ServeHTTP(w, request.WithContext(ctx))
	})
}
