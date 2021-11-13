package webserver

import (
	"context"
	transfer2 "github.com/romanornr/autodealer/internal/transfer"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"net/http"

	"github.com/go-chi/render"
	"github.com/sirupsen/logrus"
	"gopkg.in/errgo.v2/fmt/errors"
)

func bankTransferHandler(w http.ResponseWriter, r *http.Request) {
	err := tpl.ExecuteTemplate(w, "bank.html", nil)
	if err != nil {
		logrus.Errorf("error template: %s\n", err)
	}
}

func getBankTransfer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	exchangeResponse, ok := ctx.Value("response").(*transfer2.ExchangeWithdrawResponse)
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnprocessableEntity), http.StatusUnprocessableEntity)
		render.Render(w, r, transfer2.ErrWithdawRender(errors.Newf("kraken international bank account request failed CTX")))
		return
	}

	render.Render(w, r, exchangeResponse)

	return
}

func BankTransferCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		//currencyCode := currency.NewCode(chi.URLParam(request, "exchange"))     // TODO fix currency.EUR
		currencyCode := currency.EUR

		submitResponse, err := transfer2.KrakenConvertUSDT(currencyCode)
		if err != nil {
			logrus.Errorf("Failed to sell USDT to Euro: %s\n", err)
			render.Render(w, request, transfer2.ErrWithdawRender(err))
			return
		}
		logrus.Infof("submit response %v\n", submitResponse)

		response, err := transfer2.KrakenInternationalBankAccountWithdrawal(currencyCode)
		if err != nil {
			logrus.Errorf("Failed to get bank account transfer: %s\n", err)
			response.Error = err
			render.Render(w, request, transfer2.ErrWithdawRender(err))
			return
		}

		ctx := context.WithValue(request.Context(), "response", &response)
		next.ServeHTTP(w, request.WithContext(ctx))
	})
}
