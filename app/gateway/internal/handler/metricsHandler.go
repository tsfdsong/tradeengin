package handler

import (
	"net/http"

	"github.com/tsfdsong/tradeengin/app/gateway/internal/logic"
	"github.com/tsfdsong/tradeengin/app/gateway/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func metricsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewMetricsLogic(r.Context(), svcCtx)
		resp, err := l.Metrics()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
